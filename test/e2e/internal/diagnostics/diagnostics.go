/*
Copyright © contributors to CloudNativePG, established as
CloudNativePG a Series of LF Projects, LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

SPDX-License-Identifier: Apache-2.0
*/

// Package diagnostics provides helpers to capture the state of a namespace
// when an e2e spec fails, so the root cause can be inspected from the CI
// logs without needing to reproduce the failure locally.
package diagnostics

import (
	"context"
	"fmt"
	"io"
	"slices"
	"sort"

	cloudnativepgv1 "github.com/cloudnative-pg/api/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
)

// pluginContainerName is the name of the barman-cloud plugin sidecar injected
// into every PostgreSQL pod, see internal/cnpgi/operator/lifecycle.go.
const pluginContainerName = "plugin-barman-cloud"

// tailLines is the number of trailing log lines fetched from each container.
const tailLines = 200

// DumpNamespace prints diagnostic information about the given test namespace
// to the GinkgoWriter: namespace Events, Backup and Cluster statuses, pod
// container statuses, and the tail of the postgres/plugin container logs.
// It is a no-op if the current spec has not failed, so it is safe to call
// unconditionally from an AfterEach, before the namespace is torn down.
func DumpNamespace(ctx context.Context, cl client.Client, clientSet *kubernetes.Clientset, namespaceName string) {
	if !CurrentSpecReport().Failed() {
		return
	}

	dumpNamespace(ctx, cl, clientSet, namespaceName, "postgres", pluginContainerName)
}

// DumpOperatorNamespace prints the same diagnostics as DumpNamespace, but for
// the cnpg-system namespace the CloudNativePG operator and the barman-cloud
// plugin run in, dumping the logs of every container in it. Unlike
// DumpNamespace it always runs when called: the caller (typically a
// ReportAfterSuite, since cnpg-system isn't torn down per-spec) is expected
// to only call it once the aggregated suite report shows a failure.
func DumpOperatorNamespace(
	ctx context.Context,
	cl client.Client,
	clientSet *kubernetes.Clientset,
	namespaceName string,
) {
	dumpNamespace(ctx, cl, clientSet, namespaceName)
}

func dumpNamespace(
	ctx context.Context,
	cl client.Client,
	clientSet *kubernetes.Clientset,
	namespaceName string,
	containerNames ...string,
) {
	fmt.Fprintf(GinkgoWriter, "\n::group::Diagnostics for namespace %q\n", namespaceName)
	defer fmt.Fprintln(GinkgoWriter, "::endgroup::")

	dumpEvents(ctx, cl, namespaceName)
	dumpBackups(ctx, cl, namespaceName)
	dumpClusters(ctx, cl, namespaceName)
	dumpPods(ctx, cl, clientSet, namespaceName, containerNames)
}

func dumpEvents(ctx context.Context, cl client.Client, namespaceName string) {
	var events corev1.EventList
	if err := cl.List(ctx, &events, client.InNamespace(namespaceName)); err != nil {
		fmt.Fprintf(GinkgoWriter, "failed to list events in %q: %v\n", namespaceName, err)
		return
	}

	sort.Slice(events.Items, func(i, j int) bool {
		return events.Items[i].LastTimestamp.Before(&events.Items[j].LastTimestamp)
	})

	fmt.Fprintf(GinkgoWriter, "-- Events (%d) --\n", len(events.Items))
	for _, event := range events.Items {
		fmt.Fprintf(GinkgoWriter, "[%s] %s/%s %s: %s\n",
			event.LastTimestamp.Format("15:04:05"),
			event.InvolvedObject.Kind, event.InvolvedObject.Name,
			event.Reason, event.Message)
	}
}

func dumpBackups(ctx context.Context, cl client.Client, namespaceName string) {
	var backups cloudnativepgv1.BackupList
	if err := cl.List(ctx, &backups, client.InNamespace(namespaceName)); err != nil {
		fmt.Fprintf(GinkgoWriter, "failed to list backups in %q: %v\n", namespaceName, err)
		return
	}

	fmt.Fprintf(GinkgoWriter, "-- Backups (%d) --\n", len(backups.Items))
	for _, backup := range backups.Items {
		fmt.Fprintf(GinkgoWriter, "%s: phase=%s error=%q commandError=%q\n",
			backup.Name, backup.Status.Phase, backup.Status.Error, backup.Status.CommandError)
	}
}

func dumpClusters(ctx context.Context, cl client.Client, namespaceName string) {
	var clusters cloudnativepgv1.ClusterList
	if err := cl.List(ctx, &clusters, client.InNamespace(namespaceName)); err != nil {
		fmt.Fprintf(GinkgoWriter, "failed to list clusters in %q: %v\n", namespaceName, err)
		return
	}

	fmt.Fprintf(GinkgoWriter, "-- Clusters (%d) --\n", len(clusters.Items))
	for _, cluster := range clusters.Items {
		fmt.Fprintf(GinkgoWriter, "%s: phase=%s reason=%q readyInstances=%d/%d\n",
			cluster.Name, cluster.Status.Phase, cluster.Status.PhaseReason,
			cluster.Status.ReadyInstances, cluster.Spec.Instances)
	}
}

// dumpPods prints every pod's container statuses in namespaceName, and the
// tail of the log of each container whose name is in containerNames (or
// every container, if containerNames is empty).
func dumpPods(
	ctx context.Context,
	cl client.Client,
	clientSet *kubernetes.Clientset,
	namespaceName string,
	containerNames []string,
) {
	var pods corev1.PodList
	if err := cl.List(ctx, &pods, client.InNamespace(namespaceName)); err != nil {
		fmt.Fprintf(GinkgoWriter, "failed to list pods in %q: %v\n", namespaceName, err)
		return
	}

	fmt.Fprintf(GinkgoWriter, "-- Pods (%d) --\n", len(pods.Items))
	for _, pod := range pods.Items {
		fmt.Fprintf(GinkgoWriter, "%s: phase=%s\n", pod.Name, pod.Status.Phase)
		for _, cs := range pod.Status.ContainerStatuses {
			fmt.Fprintf(GinkgoWriter, "  container %s: ready=%t restarts=%d state=%s\n",
				cs.Name, cs.Ready, cs.RestartCount, containerStateString(cs.State))

			if len(containerNames) == 0 || slices.Contains(containerNames, cs.Name) {
				dumpContainerLog(ctx, clientSet, namespaceName, pod.Name, cs.Name)
			}
		}
	}
}

func containerStateString(state corev1.ContainerState) string {
	switch {
	case state.Waiting != nil:
		return fmt.Sprintf("waiting(%s: %s)", state.Waiting.Reason, state.Waiting.Message)
	case state.Running != nil:
		return fmt.Sprintf("running(since %s)", state.Running.StartedAt)
	case state.Terminated != nil:
		return fmt.Sprintf("terminated(exitCode=%d reason=%s: %s)",
			state.Terminated.ExitCode, state.Terminated.Reason, state.Terminated.Message)
	default:
		return "unknown"
	}
}

func dumpContainerLog(
	ctx context.Context,
	clientSet *kubernetes.Clientset,
	namespaceName, podName, containerName string,
) {
	tail := int64(tailLines)
	req := clientSet.CoreV1().Pods(namespaceName).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
		TailLines: &tail,
	})

	stream, err := req.Stream(ctx)
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "  failed to fetch logs for %s/%s: %v\n", podName, containerName, err)
		return
	}
	defer stream.Close()

	logs, err := io.ReadAll(stream)
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "  failed to read logs for %s/%s: %v\n", podName, containerName, err)
		return
	}

	fmt.Fprintf(GinkgoWriter, "  -- last %d lines of %s/%s --\n%s\n", tailLines, podName, containerName, logs)
}
