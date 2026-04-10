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

package credentialrotation

import (
	"time"

	cloudnativepgv1 "github.com/cloudnative-pg/api/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/specs"
	internalClient "github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/client"
	internalCluster "github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/cluster"
	nmsp "github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/namespace"
	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/objectstore"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	clusterName     = "source"
	objectStoreName = "source"
	oldSecretName   = "minio"
	newSecretName   = "minio-rotated"
)

var _ = Describe("Credential rotation", func() {
	var namespace *corev1.Namespace
	var cl client.Client

	BeforeEach(func(ctx SpecContext) {
		var err error
		cl, _, err = internalClient.NewClient()
		Expect(err).NotTo(HaveOccurred())
		namespace, err = nmsp.CreateUniqueNamespace(ctx, cl, "cred-rotation")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func(ctx SpecContext) {
		Expect(cl.Delete(ctx, namespace)).To(Succeed())
	})

	It("should update the Role when the ObjectStore secret reference changes", func(ctx SpecContext) {
		By("starting the ObjectStore deployment")
		resources := objectstore.NewMinioObjectStoreResources(namespace.Name, oldSecretName)
		Expect(resources.Create(ctx, cl)).To(Succeed())

		By("creating the ObjectStore")
		store := objectstore.NewMinioObjectStore(namespace.Name, objectStoreName, oldSecretName)
		Expect(cl.Create(ctx, store)).To(Succeed())

		By("creating the Cluster")
		cluster := newCluster(namespace.Name)
		Expect(cl.Create(ctx, cluster)).To(Succeed())

		By("waiting for the Cluster to be ready")
		Eventually(func(g Gomega) {
			g.Expect(cl.Get(ctx, types.NamespacedName{
				Name:      cluster.Name,
				Namespace: cluster.Namespace,
			}, cluster)).To(Succeed())
			g.Expect(internalCluster.IsReady(*cluster)).To(BeTrue())
		}).WithTimeout(10 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())

		roleKey := types.NamespacedName{
			Name:      specs.GetRBACName(clusterName),
			Namespace: namespace.Name,
		}

		By("verifying the Role has the cluster label and references the original secret")
		var role rbacv1.Role
		Expect(cl.Get(ctx, roleKey, &role)).To(Succeed())
		Expect(role.Labels).To(HaveKeyWithValue(metadata.ClusterLabelName, clusterName))
		Expect(secretNamesFromRole(&role)).To(ContainElement(oldSecretName))

		By("creating a new secret with the same credentials")
		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      newSecretName,
				Namespace: namespace.Name,
			},
			Data: map[string][]byte{
				"ACCESS_KEY_ID":     []byte("minio"),
				"ACCESS_SECRET_KEY": []byte("minio123"),
			},
		}
		Expect(cl.Create(ctx, newSecret)).To(Succeed())

		By("updating the ObjectStore to reference the new secret")
		Expect(cl.Get(ctx, types.NamespacedName{
			Name:      objectStoreName,
			Namespace: namespace.Name,
		}, store)).To(Succeed())
		store.Spec.Configuration.BarmanCredentials.AWS.AccessKeyIDReference.Name = newSecretName
		store.Spec.Configuration.BarmanCredentials.AWS.SecretAccessKeyReference.Name = newSecretName
		Expect(cl.Update(ctx, store)).To(Succeed())

		By("waiting for the Role to reference the new secret")
		Eventually(func(g Gomega) {
			g.Expect(cl.Get(ctx, roleKey, &role)).To(Succeed())
			g.Expect(secretNamesFromRole(&role)).To(ContainElement(newSecretName))
			g.Expect(secretNamesFromRole(&role)).NotTo(ContainElement(oldSecretName))
		}).WithTimeout(3 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
	})
})

func newCluster(namespace string) *cloudnativepgv1.Cluster {
	return &cloudnativepgv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "postgresql.cnpg.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: namespace,
		},
		Spec: cloudnativepgv1.ClusterSpec{
			Instances:       1,
			ImagePullPolicy: corev1.PullAlways,
			Plugins: []cloudnativepgv1.PluginConfiguration{
				{
					Name: "barman-cloud.cloudnative-pg.io",
					Parameters: map[string]string{
						"barmanObjectName": objectStoreName,
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

func secretNamesFromRole(role *rbacv1.Role) []string {
	for _, rule := range role.Rules {
		if len(rule.APIGroups) == 1 &&
			rule.APIGroups[0] == "" &&
			len(rule.Resources) == 1 &&
			rule.Resources[0] == "secrets" {
			return rule.ResourceNames
		}
	}

	return nil
}
