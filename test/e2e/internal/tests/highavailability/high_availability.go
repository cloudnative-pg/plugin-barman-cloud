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

package highavailability

import (
	"fmt"
	"strings"
	"time"

	cloudnativepgv1 "github.com/cloudnative-pg/api/pkg/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	internalClient "github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/client"
	internalCluster "github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/cluster"
	nmsp "github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/namespace"
	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/objectstore"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	pluginNamespace  = "cnpg-system"
	pluginDeployment = "barman-cloud"
	leaderElectionID = "822e3f5c.cnpg.io"

	objectStoreName   = "source"
	s3SecretName      = "s3"
	clusterName       = "source"
	secondClusterName = "second"
)

var _ = Describe("Plugin high availability", Serial, func() {
	var namespace *corev1.Namespace
	var cl client.Client
	var clientSet *kubernetes.Clientset

	BeforeEach(func(ctx SpecContext) {
		var err error
		cl, _, err = internalClient.NewClient()
		Expect(err).NotTo(HaveOccurred())
		clientSet, _, err = internalClient.NewClientSet()
		Expect(err).NotTo(HaveOccurred())
		namespace, err = nmsp.CreateUniqueNamespace(ctx, cl, "plugin-ha")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func(ctx SpecContext) {
		Expect(cl.Delete(ctx, namespace)).To(Succeed())
		Expect(scaleDeployment(ctx, cl, 1)).To(Succeed())
		Eventually(func(g Gomega) {
			var deploy appsv1.Deployment
			g.Expect(cl.Get(ctx, types.NamespacedName{
				Name:      pluginDeployment,
				Namespace: pluginNamespace,
			}, &deploy)).To(Succeed())
			g.Expect(deploy.Status.Replicas).To(BeEquivalentTo(1))
			g.Expect(deploy.Status.ReadyReplicas).To(BeEquivalentTo(1))
		}).WithTimeout(2 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
	})

	It("should serve requests from every replica and survive losing the leader", func(ctx SpecContext) {
		By("scaling the plugin Deployment to 2 replicas")
		Expect(scaleDeployment(ctx, cl, 2)).To(Succeed())

		By("waiting for both replicas to become ready")
		Eventually(func(g Gomega) {
			var deploy appsv1.Deployment
			g.Expect(cl.Get(ctx, types.NamespacedName{
				Name:      pluginDeployment,
				Namespace: pluginNamespace,
			}, &deploy)).To(Succeed())
			g.Expect(deploy.Status.ReadyReplicas).To(BeEquivalentTo(2))
			g.Expect(deploy.Status.UpdatedReplicas).To(BeEquivalentTo(2))
		}).WithTimeout(2 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())

		By("finding the current leader")
		leaderPodName, err := getLeaderPodName(ctx, clientSet)
		Expect(err).NotTo(HaveOccurred())
		Expect(podExists(ctx, cl, leaderPodName)).To(BeTrue())

		By("starting the ObjectStore deployment")
		resources := objectstore.NewS3ObjectStoreResources(namespace.Name, s3SecretName)
		Expect(resources.Create(ctx, cl)).To(Succeed())

		By("creating the ObjectStore")
		store := objectstore.NewS3ObjectStore(namespace.Name, objectStoreName, s3SecretName)
		Expect(cl.Create(ctx, store)).To(Succeed())

		By("creating the Cluster")
		cluster := newCluster(namespace.Name, clusterName, objectStoreName)
		Expect(cl.Create(ctx, cluster)).To(Succeed())

		By("waiting for the Cluster to be ready, exercising gRPC across both replicas")
		waitForClusterReady(ctx, cl, cluster)

		By("deleting the leader Pod")
		Expect(cl.Delete(ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      leaderPodName,
				Namespace: pluginNamespace,
			},
		})).To(Succeed())

		By("waiting for a new leader to be elected and both replicas to be ready again")
		Eventually(func(g Gomega) {
			newLeaderPodName, err := getLeaderPodName(ctx, clientSet)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(newLeaderPodName).NotTo(Equal(leaderPodName))
			g.Expect(podExists(ctx, cl, newLeaderPodName)).To(BeTrue())

			var deploy appsv1.Deployment
			g.Expect(cl.Get(ctx, types.NamespacedName{
				Name:      pluginDeployment,
				Namespace: pluginNamespace,
			}, &deploy)).To(Succeed())
			g.Expect(deploy.Status.ReadyReplicas).To(BeEquivalentTo(2))
		}).WithTimeout(3 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())

		By("verifying reconciliation still works after the failover")
		secondCluster := newCluster(namespace.Name, secondClusterName, objectStoreName)
		Expect(cl.Create(ctx, secondCluster)).To(Succeed())
		waitForClusterReady(ctx, cl, secondCluster)
	})
})

func scaleDeployment(ctx SpecContext, cl client.Client, replicas int32) error {
	var deploy appsv1.Deployment
	if err := cl.Get(ctx, types.NamespacedName{
		Name:      pluginDeployment,
		Namespace: pluginNamespace,
	}, &deploy); err != nil {
		return err //nolint:wrapcheck
	}
	deploy.Spec.Replicas = ptr.To(replicas)

	return cl.Update(ctx, &deploy) //nolint:wrapcheck
}

// getLeaderPodName reads the leader election Lease and returns the name of
// the Pod currently holding it. HolderIdentity has the form <podName>_<uuid>.
func getLeaderPodName(ctx SpecContext, clientSet *kubernetes.Clientset) (string, error) {
	lease, err := clientSet.CoordinationV1().Leases(pluginNamespace).Get(ctx, leaderElectionID, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get %s lease: %w", leaderElectionID, err)
	}
	if lease.Spec.HolderIdentity == nil || *lease.Spec.HolderIdentity == "" {
		return "", nil
	}

	return strings.Split(*lease.Spec.HolderIdentity, "_")[0], nil
}

func podExists(ctx SpecContext, cl client.Client, name string) bool {
	var pod corev1.Pod
	err := cl.Get(ctx, types.NamespacedName{Name: name, Namespace: pluginNamespace}, &pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false
		}
		Fail(err.Error())
	}

	return true
}

func waitForClusterReady(ctx SpecContext, cl client.Client, cluster *cloudnativepgv1.Cluster) {
	Eventually(func(g Gomega) {
		g.Expect(cl.Get(ctx, types.NamespacedName{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
		}, cluster)).To(Succeed())
		g.Expect(internalCluster.IsReady(*cluster)).To(BeTrue())
	}).WithTimeout(10 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())
}

func newCluster(namespace, name, objectStore string) *cloudnativepgv1.Cluster {
	return &cloudnativepgv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "postgresql.cnpg.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cloudnativepgv1.ClusterSpec{
			Instances:       1,
			ImagePullPolicy: corev1.PullAlways,
			Plugins: []cloudnativepgv1.PluginConfiguration{
				{
					Name: "barman-cloud.cloudnative-pg.io",
					Parameters: map[string]string{
						"barmanObjectName": objectStore,
					},
					IsWALArchiver: ptr.To(true),
				},
			},
			StorageConfiguration: cloudnativepgv1.StorageConfiguration{
				Size: "1Gi",
			},
		},
	}
}
