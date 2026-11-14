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

package operator

import (
	"encoding/json"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cnpg-i/pkg/reconciler"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ReconcilerImplementation", func() {
	const (
		namespace       = "test-namespace"
		clusterName     = "cluster"
		objectStoreName = "object-store"
	)

	var cluster *cnpgv1.Cluster

	newScheme := func() *runtime.Scheme {
		s := runtime.NewScheme()
		barmancloudv1.AddKnownTypes(s)
		Expect(cnpgv1.AddToScheme(s)).To(Succeed())
		Expect(rbacv1.AddToScheme(s)).To(Succeed())
		return s
	}

	newRequest := func(obj any) *reconciler.ReconcilerHooksRequest {
		definition, err := json.Marshal(obj)
		Expect(err).NotTo(HaveOccurred())
		return &reconciler.ReconcilerHooksRequest{ResourceDefinition: definition}
	}

	BeforeEach(func() {
		cluster = &cnpgv1.Cluster{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Cluster",
				APIVersion: "postgresql.cnpg.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
			},
			Spec: cnpgv1.ClusterSpec{
				Plugins: []cnpgv1.PluginConfiguration{
					{
						Name:          metadata.PluginName,
						IsWALArchiver: ptr.To(true),
						Parameters: map[string]string{
							"barmanObjectName": objectStoreName,
						},
					},
				},
			},
		}
	})

	Describe("Pre", func() {
		It("asks to be called again after a delay when the ObjectStore does not exist yet", func(ctx SpecContext) {
			r := ReconcilerImplementation{
				Client: fake.NewClientBuilder().WithScheme(newScheme()).Build(),
			}

			result, err := r.Pre(ctx, newRequest(cluster))
			Expect(err).NotTo(HaveOccurred())
			Expect(result.GetBehavior()).To(Equal(reconciler.ReconcilerHooksResult_BEHAVIOR_REQUEUE))
			// Without a delay the operator does not requeue at all, and the
			// Cluster is never reconciled again (issue #1236).
			Expect(result.GetRequeueAfter()).To(BeNumerically(">", 0))
			Expect(result.GetRequeueAfter()).To(Equal(objectStoreNotFoundRequeueAfter))
		})

		It("continues once the ObjectStore exists", func(ctx SpecContext) {
			store := &barmancloudv1.ObjectStore{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ObjectStore",
					APIVersion: barmancloudv1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      objectStoreName,
					Namespace: namespace,
				},
			}
			r := ReconcilerImplementation{
				Client: fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(store).Build(),
			}

			result, err := r.Pre(ctx, newRequest(cluster))
			Expect(err).NotTo(HaveOccurred())
			Expect(result.GetBehavior()).To(Equal(reconciler.ReconcilerHooksResult_BEHAVIOR_CONTINUE))
			Expect(result.GetRequeueAfter()).To(BeZero())
		})

		It("continues without looking up ObjectStores for kinds other than Cluster", func(ctx SpecContext) {
			r := ReconcilerImplementation{
				Client: fake.NewClientBuilder().WithScheme(newScheme()).Build(),
			}
			backup := &cnpgv1.Backup{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Backup",
					APIVersion: "postgresql.cnpg.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "backup",
					Namespace: namespace,
				},
			}

			result, err := r.Pre(ctx, newRequest(backup))
			Expect(err).NotTo(HaveOccurred())
			Expect(result.GetBehavior()).To(Equal(reconciler.ReconcilerHooksResult_BEHAVIOR_CONTINUE))
		})
	})
})
