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

package common

import (
	"context"
	"os"
	"path/filepath"

	barmanapi "github.com/cloudnative-pg/barman-cloud/pkg/api"
	barmanRestorer "github.com/cloudnative-pg/barman-cloud/pkg/restorer"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("resolveRestoreObjectStore", func() {
	const (
		namespace = "test-ns"
		instance  = "cluster-1"
	)

	// newConfig builds a PluginConfiguration with distinct, recognizable names
	// for every candidate object store, so each test can assert exactly which
	// one the routing selected.
	newConfig := func(currentPrimary, replicaSourceObject string) *config.PluginConfiguration {
		return &config.PluginConfiguration{
			Cluster: &cnpgv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
				Status:     cnpgv1.ClusterStatus{CurrentPrimary: currentPrimary},
			},
			BarmanObjectName:              "cluster-store",
			ServerName:                    "cluster-server",
			RecoveryBarmanObjectName:      "recovery-store",
			RecoveryServerName:            "recovery-server",
			ReplicaSourceBarmanObjectName: replicaSourceObject,
			ReplicaSourceServerName:       "replica-server",
		}
	}

	DescribeTable(
		"selects the correct object store for restoring WAL files",
		func(cfg *config.PluginConfiguration, wantServer, wantObject string) {
			gotServer, gotKey := resolveRestoreObjectStore(cfg, instance)

			Expect(gotServer).To(Equal(wantServer))
			Expect(gotKey.Name).To(Equal(wantObject))
			Expect(gotKey.Namespace).To(Equal(namespace))
		},

		// The regression this guards: during a designated-primary promotion the
		// instance is already the current primary while still in recovery, and it
		// must pull remaining WALs from the replica source. The routing decision does
		// not depend on the promotion token, so this single case covers both
		// switchover and failover.
		Entry("designated primary in promotion -> replica source",
			newConfig(instance, "replica-store"),
			"replica-server", "replica-store"),

		// Guards the len(ReplicaSourceBarmanObjectName) > 0 gate: a current primary
		// without a barman-backed replica source (plain HA primary, or a replica
		// cluster whose source is streaming-only) must use the cluster store, not
		// an empty-named replica source key.
		Entry("current primary without a replica source -> cluster store",
			newConfig(instance, ""),
			"cluster-server", "cluster-store"),

		// Bootstrap / PITR: no current primary yet. Recovery wins even if a replica
		// source happens to be configured.
		Entry("no current primary -> recovery store",
			newConfig("", "replica-store"),
			"recovery-server", "recovery-store"),

		Entry("ordinary standby -> cluster store",
			newConfig("cluster-2", ""),
			"cluster-server", "cluster-store"),

		// A non-primary instance must never route to the replica source, even when
		// one is configured: only the designated primary catches up from the source.
		Entry("standby in a replica cluster -> cluster store",
			newConfig("cluster-2", "replica-store"),
			"cluster-server", "cluster-store"),
	)
})

var _ = Describe("maxWALFilesPerInvocation", func() {
	configWithMaxParallel := func(maxParallel int) *barmanapi.BarmanObjectStoreConfiguration {
		return &barmanapi.BarmanObjectStoreConfiguration{
			Wal: &barmanapi.WalBackupConfiguration{MaxParallel: maxParallel},
		}
	}

	DescribeTable(
		"computes how many WAL files a single invocation may fetch",
		func(cfg *barmanapi.BarmanObjectStoreConfiguration, rewindMode bool, want int) {
			Expect(maxWALFilesPerInvocation(cfg, rewindMode)).To(Equal(want))
		},

		Entry("no WAL configuration", &barmanapi.BarmanObjectStoreConfiguration{}, false, 1),
		Entry("parallel restore configured", configWithMaxParallel(8), false, 8),

		// pg_rewind walks the timeline backwards: prefetching must stay off no
		// matter what the object store configuration asks for
		Entry("rewind mode overrides the configured parallelism", configWithMaxParallel(8), true, 1),
	)
})

var _ = Describe("shouldUseEndOfWALStreamFlag", func() {
	clusterWithPrimary := func(currentPrimary string) *cnpgv1.Cluster {
		return &cnpgv1.Cluster{
			Status: cnpgv1.ClusterStatus{CurrentPrimary: currentPrimary},
		}
	}

	DescribeTable(
		"decides whether the end-of-wal-stream flag machinery applies",
		func(cluster *cnpgv1.Cluster, podName string, rewindMode bool, want bool) {
			Expect(shouldUseEndOfWALStreamFlag(cluster, podName, rewindMode)).To(Equal(want))
		},

		Entry("replica with streaming available", clusterWithPrimary("cluster-1"), "cluster-2", false, true),
		Entry("primary cannot stream from anyone", clusterWithPrimary("cluster-1"), "cluster-1", false, false),

		// pg_rewind cannot fall back to streaming replication: the flag machinery
		// must stay off even where a standby would use it
		Entry("rewind mode wins over streaming availability", clusterWithPrimary("cluster-1"), "cluster-2", true, false),
	)
})

var _ = Describe("clearEndOfWALStreamFlag", func() {
	newRestorer := func() *barmanRestorer.WALRestorer {
		restorer, err := barmanRestorer.New(context.Background(), nil, GinkgoT().TempDir())
		Expect(err).ToNot(HaveOccurred())
		return restorer
	}

	It("is a no-op when the flag is not set", func() {
		restorer := newRestorer()

		Expect(clearEndOfWALStreamFlag(restorer)).To(Succeed())

		isEOS, err := restorer.IsEndOfWALStream()
		Expect(err).ToNot(HaveOccurred())
		Expect(isEOS).To(BeFalse())
	})

	// Regression guard: a flag left over from a normal-recovery invocation that
	// ran before this pod was demoted must not survive a pg_rewind restore, or
	// it would resurface and wrongly abort the first normal-recovery invocation
	// that runs once the rewind is done.
	It("removes a pre-existing flag without returning an error", func() {
		restorer := newRestorer()
		Expect(restorer.SetEndOfWALStream()).To(Succeed())

		Expect(clearEndOfWALStreamFlag(restorer)).To(Succeed())

		isEOS, err := restorer.IsEndOfWALStream()
		Expect(err).ToNot(HaveOccurred())
		Expect(isEOS).To(BeFalse())
	})
})

var _ = Describe("resolveArchiveEmptyWalArchiveCheck", func() {
	// skipAnnotation mirrors the unexported constant in cloudnative-pg's
	// pkg/utils; hard-coding the literal makes a divergence surface as a
	// failing test rather than silently disabling the check.
	const skipAnnotation = "cnpg.io/skipEmptyWalArchiveCheck"

	clusterWith := func(annotationValue *string) *cnpgv1.Cluster {
		cluster := &cnpgv1.Cluster{}
		if annotationValue != nil {
			cluster.Annotations = map[string]string{skipAnnotation: *annotationValue}
		}
		return cluster
	}

	// markerPath returns the marker file path inside a fresh temp dir,
	// creating the file there when present is true.
	markerPath := func(present bool) string {
		filePath := filepath.Join(GinkgoT().TempDir(), metadata.CheckEmptyWalArchiveFile)
		if present {
			Expect(os.WriteFile(filePath, []byte{}, 0o600)).To(Succeed())
		}
		return filePath
	}

	When("the operator sets the decision", func() {
		It("obeys true, ignoring the annotation and the marker file", func() {
			// annotation would skip the check and the marker is absent, yet the
			// operator's explicit true must still win.
			got, err := resolveArchiveEmptyWalArchiveCheck(
				ptr.To(true), clusterWith(ptr.To("enabled")), markerPath(false))
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(BeTrue())
		})

		It("obeys false, ignoring the annotation and the marker file", func() {
			// annotation would keep the check on and the marker is present, yet the
			// operator's explicit false must still win.
			got, err := resolveArchiveEmptyWalArchiveCheck(
				ptr.To(false), clusterWith(nil), markerPath(true))
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(BeFalse())
		})
	})

	When("the operator predates the field (nil decision)", func() {
		DescribeTable(
			"falls back to the annotation combined with the marker file",
			func(annotationValue *string, markerPresent bool, expected bool) {
				got, err := resolveArchiveEmptyWalArchiveCheck(
					nil, clusterWith(annotationValue), markerPath(markerPresent))
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(Equal(expected))
			},
			Entry("no annotation and marker present: check runs", nil, true, true),
			Entry("no annotation and marker absent: check skipped", nil, false, false),
			Entry("opt-out annotation and marker present: check skipped", ptr.To("enabled"), true, false),
			Entry("unrelated annotation value and marker present: check runs", ptr.To("something-else"), true, true),
			Entry("empty annotation value and marker present: check runs", ptr.To(""), true, true),
		)
	})
})
