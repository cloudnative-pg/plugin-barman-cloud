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

package objectstoreorder

import (
	"time"

	cloudnativepgv1 "github.com/cloudnative-pg/api/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	objectStoreName = "store"
	s3SecretName    = "s3"
	clusterName     = "cluster"
)

var _ = Describe("Cluster created before its ObjectStore", func() {
	var namespace *corev1.Namespace
	var cl client.Client

	BeforeEach(func(ctx SpecContext) {
		var err error
		cl, _, err = internalClient.NewClient()
		Expect(err).NotTo(HaveOccurred())
		namespace, err = nmsp.CreateUniqueNamespace(ctx, cl, "objectstore-order")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func(ctx SpecContext) {
		Expect(cl.Delete(ctx, namespace)).To(Succeed())
	})

	It("becomes ready once the ObjectStore is created, without any change to the Cluster", func(ctx SpecContext) {
		By("starting the object storage server")
		resources := objectstore.NewS3ObjectStoreResources(namespace.Name, s3SecretName)
		Expect(resources.Create(ctx, cl)).To(Succeed())

		By("creating the Cluster before the ObjectStore it refers to")
		cluster := newCluster(namespace.Name, clusterName, objectStoreName)
		Expect(cl.Create(ctx, cluster)).To(Succeed())
		createdGeneration := cluster.Generation

		By("checking that no instance is created while the ObjectStore is missing")
		Consistently(func(g Gomega) {
			var pods corev1.PodList
			g.Expect(cl.List(ctx, &pods,
				client.InNamespace(namespace.Name),
				client.MatchingLabels{"cnpg.io/cluster": clusterName},
			)).To(Succeed())
			g.Expect(pods.Items).To(BeEmpty())
		}).WithTimeout(30 * time.Second).WithPolling(5 * time.Second).Should(Succeed())

		By("creating the ObjectStore")
		store := objectstore.NewS3ObjectStore(namespace.Name, objectStoreName, s3SecretName)
		Expect(cl.Create(ctx, store)).To(Succeed())

		By("waiting for the Cluster to become ready on its own")
		Eventually(func(g Gomega) {
			g.Expect(cl.Get(ctx, types.NamespacedName{
				Name:      cluster.Name,
				Namespace: cluster.Namespace,
			}, cluster)).To(Succeed())
			g.Expect(internalCluster.IsReady(*cluster)).To(BeTrue())
		}).WithTimeout(10 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())

		By("checking that the Cluster spec was not changed to get it reconciled")
		Expect(cluster.Generation).To(Equal(createdGeneration))
	})
})

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
