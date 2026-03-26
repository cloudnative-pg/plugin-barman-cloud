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

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	barmanapi "github.com/cloudnative-pg/barman-cloud/pkg/api"
	machineryapi "github.com/cloudnative-pg/machinery/pkg/api"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
)

func newFakeScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = rbacv1.AddToScheme(s)
	_ = cnpgv1.AddToScheme(s)
	_ = barmancloudv1.AddToScheme(s)
	return s
}

func newTestCluster(name, namespace, objectStoreName string) *cnpgv1.Cluster {
	return &cnpgv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cnpgv1.ClusterSpec{
			Plugins: []cnpgv1.PluginConfiguration{
				{
					Name: metadata.PluginName,
					Parameters: map[string]string{
						"barmanObjectName": objectStoreName,
					},
				},
			},
		},
	}
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

var _ = Describe("referencesObjectStore", func() {
	It("should return true when ObjectStore is in the list", func() {
		refs := []client.ObjectKey{
			{Name: "store-a", Namespace: "default"},
			{Name: "store-b", Namespace: "default"},
		}
		Expect(referencesObjectStore(refs, client.ObjectKey{
			Name: "store-b", Namespace: "default",
		})).To(BeTrue())
	})

	It("should return false when ObjectStore is not in the list", func() {
		refs := []client.ObjectKey{
			{Name: "store-a", Namespace: "default"},
		}
		Expect(referencesObjectStore(refs, client.ObjectKey{
			Name: "store-b", Namespace: "default",
		})).To(BeFalse())
	})

	It("should return false when namespace differs", func() {
		refs := []client.ObjectKey{
			{Name: "store-a", Namespace: "ns1"},
		}
		Expect(referencesObjectStore(refs, client.ObjectKey{
			Name: "store-a", Namespace: "ns2",
		})).To(BeFalse())
	})

	It("should return false for empty list", func() {
		Expect(referencesObjectStore(nil, client.ObjectKey{
			Name: "store-a", Namespace: "default",
		})).To(BeFalse())
	})
})

var _ = Describe("ObjectStoreReconciler", func() {
	var (
		ctx        context.Context
		scheme     *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = newFakeScheme()
	})

	Describe("Reconcile", func() {
		It("should create a Role for a Cluster that references the ObjectStore", func() {
			objectStore := newTestObjectStore("my-store", "default", "aws-creds")
			cluster := newTestCluster("my-cluster", "default", "my-store")

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objectStore, cluster).
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

			var role rbacv1.Role
			err = fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "my-cluster-barman-cloud",
			}, &role)
			Expect(err).NotTo(HaveOccurred())
			Expect(role.Rules).To(HaveLen(3))

			// Verify the secrets rule contains the expected secret
			secretsRule := role.Rules[2]
			Expect(secretsRule.ResourceNames).To(ContainElement("aws-creds"))

			// Verify owner reference is set to the Cluster
			Expect(role.OwnerReferences).To(HaveLen(1))
			Expect(role.OwnerReferences[0].Name).To(Equal("my-cluster"))
			Expect(role.OwnerReferences[0].Kind).To(Equal("Cluster"))
		})

		It("should skip Clusters that don't reference the ObjectStore", func() {
			objectStore := newTestObjectStore("my-store", "default", "aws-creds")
			cluster := newTestCluster("my-cluster", "default", "other-store")

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objectStore, cluster).
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

			// No Role should have been created
			var role rbacv1.Role
			err = fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "my-cluster-barman-cloud",
			}, &role)
			Expect(err).To(HaveOccurred())
		})

		It("should succeed with no Clusters in the namespace", func() {
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
	})

	Describe("reconcileRBACForCluster", func() {
		It("should skip deleted ObjectStores and still reconcile the Role", func() {
			// Cluster references two ObjectStores, but one is deleted
			cluster := newTestCluster("my-cluster", "default", "store-a")
			existingStore := newTestObjectStore("store-a", "default", "aws-creds")

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(existingStore).
				Build()

			reconciler := &ObjectStoreReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			// Pass two keys, but "store-b" doesn't exist
			err := reconciler.reconcileRBACForCluster(ctx, cluster, []client.ObjectKey{
				{Name: "store-a", Namespace: "default"},
				{Name: "store-b", Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())

			// Role should be created with only store-a's secrets
			var role rbacv1.Role
			err = fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "my-cluster-barman-cloud",
			}, &role)
			Expect(err).NotTo(HaveOccurred())
			Expect(role.Rules).To(HaveLen(3))

			// ObjectStore rule should only reference store-a
			objectStoreRule := role.Rules[0]
			Expect(objectStoreRule.ResourceNames).To(ContainElement("store-a"))
			Expect(objectStoreRule.ResourceNames).NotTo(ContainElement("store-b"))

			// Verify owner reference is set
			Expect(role.OwnerReferences).To(HaveLen(1))
			Expect(role.OwnerReferences[0].Name).To(Equal("my-cluster"))
		})

		It("should update Role when ObjectStore credentials change", func() {
			cluster := newTestCluster("my-cluster", "default", "my-store")
			oldStore := newTestObjectStore("my-store", "default", "old-secret")

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(oldStore).
				Build()

			reconciler := &ObjectStoreReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			// First reconcile - creates Role with old-secret
			err := reconciler.reconcileRBACForCluster(ctx, cluster, []client.ObjectKey{
				{Name: "my-store", Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())

			// Update the ObjectStore with new credentials
			var currentStore barmancloudv1.ObjectStore
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Name: "my-store", Namespace: "default",
			}, &currentStore)).To(Succeed())
			currentStore.Spec.Configuration.BarmanCredentials.AWS.AccessKeyIDReference.LocalObjectReference.Name = "new-secret"
			Expect(fakeClient.Update(ctx, &currentStore)).To(Succeed())

			// Second reconcile - should patch Role with new-secret
			err = reconciler.reconcileRBACForCluster(ctx, cluster, []client.ObjectKey{
				{Name: "my-store", Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())

			var role rbacv1.Role
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "my-cluster-barman-cloud",
			}, &role)).To(Succeed())

			secretsRule := role.Rules[2]
			Expect(secretsRule.ResourceNames).To(ContainElement("new-secret"))
			Expect(secretsRule.ResourceNames).NotTo(ContainElement("old-secret"))
		})
	})
})
