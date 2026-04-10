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

package controller

import (
	"context"
	"fmt"

	barmanapi "github.com/cloudnative-pg/barman-cloud/pkg/api"
	machineryapi "github.com/cloudnative-pg/machinery/pkg/api"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/specs"
)

func newFakeScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	utilruntime.Must(rbacv1.AddToScheme(s))
	utilruntime.Must(barmancloudv1.AddToScheme(s))
	return s
}

func newTestObjectStore(name, namespace, secretName string) *barmancloudv1.ObjectStore {
	return &barmancloudv1.ObjectStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: barmancloudv1.ObjectStoreSpec{
			Configuration: barmanapi.BarmanObjectStoreConfiguration{
				DestinationPath: "s3://bucket/path",
				BarmanCredentials: barmanapi.BarmanCredentials{
					AWS: &barmanapi.S3Credentials{
						AccessKeyIDReference: &machineryapi.SecretKeySelector{
							LocalObjectReference: machineryapi.LocalObjectReference{
								Name: secretName,
							},
							Key: "ACCESS_KEY_ID",
						},
					},
				},
			},
		},
	}
}

func newLabeledRole(clusterName, namespace string, objectStores []barmancloudv1.ObjectStore) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      specs.GetRBACName(clusterName),
			Namespace: namespace,
			Labels: map[string]string{
				metadata.ClusterLabelName: clusterName,
			},
		},
		Rules: specs.BuildRoleRules(objectStores),
	}
}

var _ = Describe("ObjectStoreReconciler", func() {
	var (
		ctx    context.Context
		scheme *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = newFakeScheme()
	})

	Describe("Reconcile", func() {
		It("should update Role rules when ObjectStore credentials change", func() {
			oldStore := newTestObjectStore("my-store", "default", "old-secret")
			role := newLabeledRole("my-cluster", "default", []barmancloudv1.ObjectStore{*oldStore})

			// Update the ObjectStore with new credentials
			newStore := newTestObjectStore("my-store", "default", "new-secret")

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(role, newStore).
				Build()

			reconciler := &ObjectStoreReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "my-store",
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))

			var updatedRole rbacv1.Role
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "my-cluster-barman-cloud",
			}, &updatedRole)).To(Succeed())

			secretsRule := updatedRole.Rules[2]
			Expect(secretsRule.ResourceNames).To(ContainElement("new-secret"))
			Expect(secretsRule.ResourceNames).NotTo(ContainElement("old-secret"))
		})

		It("should skip Roles that don't reference the ObjectStore", func() {
			otherStore := newTestObjectStore("other-store", "default", "other-creds")
			role := newLabeledRole("my-cluster", "default", []barmancloudv1.ObjectStore{*otherStore})

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(role).
				Build()

			reconciler := &ObjectStoreReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			var before rbacv1.Role
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "my-cluster-barman-cloud",
			}, &before)).To(Succeed())

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "unrelated-store",
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))

			var after rbacv1.Role
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "my-cluster-barman-cloud",
			}, &after)).To(Succeed())

			Expect(after.ResourceVersion).To(Equal(before.ResourceVersion))
		})

		It("should succeed with no labeled Roles in the namespace", func() {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			reconciler := &ObjectStoreReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "my-store",
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})

		It("should handle deleted ObjectStores gracefully", func() {
			storeA := newTestObjectStore("store-a", "default", "secret-a")
			storeB := newTestObjectStore("store-b", "default", "secret-b")
			role := newLabeledRole("my-cluster", "default", []barmancloudv1.ObjectStore{*storeA, *storeB})

			// Only store-a exists; store-b was deleted
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(role, storeA).
				Build()

			reconciler := &ObjectStoreReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "store-b",
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))

			var updatedRole rbacv1.Role
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "my-cluster-barman-cloud",
			}, &updatedRole)).To(Succeed())

			objectStoreRule := updatedRole.Rules[0]
			Expect(objectStoreRule.ResourceNames).To(ContainElement("store-a"))
			Expect(objectStoreRule.ResourceNames).NotTo(ContainElement("store-b"))
		})

		It("should not panic on a Role with empty rules", func() {
			emptyRole := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-barman-cloud",
					Namespace: "default",
					Labels: map[string]string{
						metadata.ClusterLabelName: "empty",
					},
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(emptyRole).
				Build()

			reconciler := &ObjectStoreReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "my-store",
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})

		It("should produce empty ResourceNames when all ObjectStores are deleted", func() {
			store := newTestObjectStore("my-store", "default", "aws-creds")
			role := newLabeledRole("my-cluster", "default", []barmancloudv1.ObjectStore{*store})

			// Don't add the ObjectStore to the fake client (simulates deletion)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(role).
				Build()

			reconciler := &ObjectStoreReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "my-store",
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))

			var updatedRole rbacv1.Role
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "my-cluster-barman-cloud",
			}, &updatedRole)).To(Succeed())

			// All rules should have empty ResourceNames
			Expect(updatedRole.Rules[0].ResourceNames).To(BeEmpty())
			Expect(updatedRole.Rules[1].ResourceNames).To(BeEmpty())
			Expect(updatedRole.Rules[2].ResourceNames).To(BeEmpty())
		})

		It("should return an error when listing Roles fails", func() {
			internalErr := apierrs.NewInternalError(fmt.Errorf("etcd timeout"))
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
						if _, ok := list.(*rbacv1.RoleList); ok {
							return internalErr
						}
						return c.List(ctx, list, opts...)
					},
				}).
				Build()

			reconciler := &ObjectStoreReconciler{Client: fakeClient, Scheme: scheme}
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "my-store", Namespace: "default"},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("while listing roles"))
		})

		It("should return an error when fetching an ObjectStore fails with a transient error", func() {
			store := newTestObjectStore("my-store", "default", "aws-creds")
			role := newLabeledRole("my-cluster", "default", []barmancloudv1.ObjectStore{*store})

			internalErr := apierrs.NewInternalError(fmt.Errorf("etcd timeout"))
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(role).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
						if _, ok := obj.(*barmancloudv1.ObjectStore); ok {
							return internalErr
						}
						return c.Get(ctx, key, obj, opts...)
					},
				}).
				Build()

			reconciler := &ObjectStoreReconciler{Client: fakeClient, Scheme: scheme}
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "my-store", Namespace: "default"},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("while reconciling role"))
		})

		It("should reconcile multiple Roles referencing the same ObjectStore", func() {
			store := newTestObjectStore("shared-store", "default", "new-secret")
			oldStore := barmancloudv1.ObjectStore{
				ObjectMeta: metav1.ObjectMeta{Name: "shared-store", Namespace: "default"},
				Spec: barmancloudv1.ObjectStoreSpec{
					Configuration: barmanapi.BarmanObjectStoreConfiguration{
						DestinationPath: "s3://bucket/path",
						BarmanCredentials: barmanapi.BarmanCredentials{
							AWS: &barmanapi.S3Credentials{
								AccessKeyIDReference: &machineryapi.SecretKeySelector{
									LocalObjectReference: machineryapi.LocalObjectReference{Name: "old-secret"},
									Key:                  "ACCESS_KEY_ID",
								},
							},
						},
					},
				},
			}

			role1 := newLabeledRole("cluster-1", "default", []barmancloudv1.ObjectStore{oldStore})
			role2 := newLabeledRole("cluster-2", "default", []barmancloudv1.ObjectStore{oldStore})

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(role1, role2, store).
				Build()

			reconciler := &ObjectStoreReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "shared-store",
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))

			for _, clusterName := range []string{"cluster-1", "cluster-2"} {
				var updatedRole rbacv1.Role
				Expect(fakeClient.Get(ctx, client.ObjectKey{
					Namespace: "default",
					Name:      specs.GetRBACName(clusterName),
				}, &updatedRole)).To(Succeed())

				secretsRule := updatedRole.Rules[2]
				Expect(secretsRule.ResourceNames).To(ContainElement("new-secret"))
				Expect(secretsRule.ResourceNames).NotTo(ContainElement("old-secret"))
			}
		})
	})
})
