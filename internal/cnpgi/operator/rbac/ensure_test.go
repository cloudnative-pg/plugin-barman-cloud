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

package rbac_test

import (
	"context"
	"fmt"

	barmanapi "github.com/cloudnative-pg/barman-cloud/pkg/api"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/utils"
	machineryapi "github.com/cloudnative-pg/machinery/pkg/api"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/rbac"
)

func expectRequiredLabels(labels map[string]string, clusterName string) {
	ExpectWithOffset(1, labels).To(HaveKeyWithValue(metadata.ClusterLabelName, clusterName))
	ExpectWithOffset(1, labels).To(HaveKeyWithValue(utils.KubernetesAppLabelName, metadata.AppLabelValue))
	ExpectWithOffset(1, labels).To(HaveKeyWithValue(utils.KubernetesAppInstanceLabelName, clusterName))
	ExpectWithOffset(1, labels).To(HaveKeyWithValue(utils.KubernetesAppManagedByLabelName, metadata.ManagedByLabelValue))
	ExpectWithOffset(1, labels).To(HaveKeyWithValue(utils.KubernetesAppComponentLabelName, utils.DatabaseComponentName))
	ExpectWithOffset(1, labels).To(HaveKeyWithValue(utils.KubernetesAppVersionLabelName, metadata.Data.Version))
}

// newPatchCountingClient returns a fake client plus a pointer to a
// counter incremented on every Patch call. Useful for asserting
// that a "no-op" reconcile path issues no Patch — more reliable
// than comparing ResourceVersion across reads, which depends on
// fake-client RV-bumping semantics that are not part of the
// controller-runtime contract.
func newPatchCountingClient(initObjs ...client.Object) (client.Client, *int) {
	count := 0
	c := fake.NewClientBuilder().
		WithScheme(newScheme()).
		WithObjects(initObjs...).
		WithInterceptorFuncs(interceptor.Funcs{
			Patch: func(
				ctx context.Context,
				client client.WithWatch,
				obj client.Object,
				patch client.Patch,
				opts ...client.PatchOption,
			) error {
				count++
				return client.Patch(ctx, obj, patch, opts...)
			},
		}).
		Build()
	return c, &count
}

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	utilruntime.Must(rbacv1.AddToScheme(s))
	utilruntime.Must(cnpgv1.AddToScheme(s))
	utilruntime.Must(barmancloudv1.AddToScheme(s))
	return s
}

func newCluster(name, namespace string) *cnpgv1.Cluster {
	return &cnpgv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: cnpgv1.SchemeGroupVersion.String(),
			Kind:       cnpgv1.ClusterKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func newObjectStore(name, namespace, secretName string) barmancloudv1.ObjectStore {
	return barmancloudv1.ObjectStore{
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

var _ = Describe("EnsureRole", func() {
	var (
		ctx        context.Context
		cluster    *cnpgv1.Cluster
		objects    []barmancloudv1.ObjectStore
		fakeClient client.Client
	)

	BeforeEach(func() {
		ctx = context.Background()
		cluster = newCluster("test-cluster", "default")
		objects = []barmancloudv1.ObjectStore{
			newObjectStore("my-store", "default", "aws-creds"),
		}
	})

	Context("when the Role does not exist", func() {
		BeforeEach(func() {
			fakeClient = fake.NewClientBuilder().WithScheme(newScheme()).Build()
		})

		It("should create the Role with owner reference and label", func() {
			err := rbac.EnsureRole(ctx, fakeClient, cluster, objects)
			Expect(err).NotTo(HaveOccurred())

			var role rbacv1.Role
			err = fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "test-cluster-barman-cloud",
			}, &role)
			Expect(err).NotTo(HaveOccurred())
			Expect(role.Rules).To(HaveLen(3))

			Expect(role.OwnerReferences).To(HaveLen(1))
			Expect(role.OwnerReferences[0].Name).To(Equal("test-cluster"))
			Expect(role.OwnerReferences[0].Kind).To(Equal("Cluster"))

			expectRequiredLabels(role.Labels, "test-cluster")
		})
	})

	Context("when the Role exists with matching rules", func() {
		var patchCount *int

		BeforeEach(func() {
			fakeClient, patchCount = newPatchCountingClient()
			Expect(rbac.EnsureRole(ctx, fakeClient, cluster, objects)).To(Succeed())
			*patchCount = 0
		})

		It("should not patch the Role", func() {
			err := rbac.EnsureRole(ctx, fakeClient, cluster, objects)
			Expect(err).NotTo(HaveOccurred())
			Expect(*patchCount).To(BeZero())
		})
	})

	Context("when the Role exists with different rules", func() {
		BeforeEach(func() {
			fakeClient = fake.NewClientBuilder().WithScheme(newScheme()).Build()
			oldObjects := []barmancloudv1.ObjectStore{
				newObjectStore("my-store", "default", "old-secret"),
			}
			Expect(rbac.EnsureRole(ctx, fakeClient, cluster, oldObjects)).To(Succeed())
		})

		It("should patch the Role with new rules and preserve owner reference", func() {
			err := rbac.EnsureRole(ctx, fakeClient, cluster, objects)
			Expect(err).NotTo(HaveOccurred())

			var role rbacv1.Role
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "test-cluster-barman-cloud",
			}, &role)).To(Succeed())

			secretsRule := role.Rules[2]
			Expect(secretsRule.ResourceNames).To(ContainElement("aws-creds"))
			Expect(secretsRule.ResourceNames).NotTo(ContainElement("old-secret"))

			Expect(role.OwnerReferences).To(HaveLen(1))
			Expect(role.OwnerReferences[0].Name).To(Equal("test-cluster"))
		})
	})

	Context("when Role creation fails with a transient error", func() {
		BeforeEach(func() {
			internalErr := apierrs.NewInternalError(fmt.Errorf("etcd timeout"))
			fakeClient = fake.NewClientBuilder().
				WithScheme(newScheme()).
				WithInterceptorFuncs(interceptor.Funcs{
					Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
						return internalErr
					},
				}).
				Build()
		})

		It("should propagate the error", func() {
			err := rbac.EnsureRole(ctx, fakeClient, cluster, objects)
			Expect(err).To(HaveOccurred())
			Expect(apierrs.IsInternalError(err)).To(BeTrue())
		})
	})

	Context("when the Role has pre-existing unrelated labels", func() {
		BeforeEach(func() {
			fakeClient = fake.NewClientBuilder().WithScheme(newScheme()).Build()
			existing := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-barman-cloud",
					Namespace: "default",
					Labels: map[string]string{
						"custom-label": "custom-value",
					},
				},
			}
			Expect(fakeClient.Create(ctx, existing)).To(Succeed())
		})

		It("should preserve unrelated labels while adding the cluster label", func() {
			err := rbac.EnsureRole(ctx, fakeClient, cluster, objects)
			Expect(err).NotTo(HaveOccurred())

			var role rbacv1.Role
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "test-cluster-barman-cloud",
			}, &role)).To(Succeed())

			Expect(role.Labels).To(HaveKeyWithValue("custom-label", "custom-value"))
			expectRequiredLabels(role.Labels, "test-cluster")
		})
	})

	Context("when the Role exists with a stale label value", func() {
		BeforeEach(func() {
			fakeClient = fake.NewClientBuilder().WithScheme(newScheme()).Build()
			existing := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-barman-cloud",
					Namespace: "default",
					Labels: map[string]string{
						// Stale value as if written by an older plugin
						// version.
						utils.KubernetesAppVersionLabelName: "0.0.0-stale",
					},
				},
			}
			Expect(fakeClient.Create(ctx, existing)).To(Succeed())
		})

		It("should overwrite the stale value with the current plugin's value", func() {
			err := rbac.EnsureRole(ctx, fakeClient, cluster, objects)
			Expect(err).NotTo(HaveOccurred())

			var role rbacv1.Role
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "test-cluster-barman-cloud",
			}, &role)).To(Succeed())

			Expect(role.Labels).To(HaveKeyWithValue(
				utils.KubernetesAppVersionLabelName, metadata.Data.Version))
		})
	})

	Context("when the Role exists without the cluster label (upgrade scenario)", func() {
		BeforeEach(func() {
			fakeClient = fake.NewClientBuilder().WithScheme(newScheme()).Build()

			// Create a Role without the label (simulates pre-upgrade state)
			unlabeledRole := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-barman-cloud",
					Namespace: "default",
				},
				Rules: []rbacv1.PolicyRule{},
			}
			Expect(fakeClient.Create(ctx, unlabeledRole)).To(Succeed())
		})

		It("should add the label and update rules", func() {
			err := rbac.EnsureRole(ctx, fakeClient, cluster, objects)
			Expect(err).NotTo(HaveOccurred())

			var role rbacv1.Role
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "test-cluster-barman-cloud",
			}, &role)).To(Succeed())

			expectRequiredLabels(role.Labels, "test-cluster")
			Expect(role.Rules).To(HaveLen(3))
		})
	})
})

var _ = Describe("EnsureRoleBinding", func() {
	var (
		ctx        context.Context
		cluster    *cnpgv1.Cluster
		fakeClient client.Client
	)

	BeforeEach(func() {
		ctx = context.Background()
		cluster = newCluster("test-cluster", "default")
	})

	Context("when the RoleBinding does not exist", func() {
		BeforeEach(func() {
			fakeClient = fake.NewClientBuilder().WithScheme(newScheme()).Build()
		})

		It("should create the RoleBinding with owner reference, labels, and correct subjects", func() {
			err := rbac.EnsureRoleBinding(ctx, fakeClient, cluster)
			Expect(err).NotTo(HaveOccurred())

			var rb rbacv1.RoleBinding
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "test-cluster-barman-cloud",
			}, &rb)).To(Succeed())

			Expect(rb.OwnerReferences).To(HaveLen(1))
			Expect(rb.OwnerReferences[0].Name).To(Equal("test-cluster"))
			Expect(rb.OwnerReferences[0].Kind).To(Equal("Cluster"))

			expectRequiredLabels(rb.Labels, "test-cluster")

			Expect(rb.Subjects).To(HaveLen(1))
			Expect(rb.Subjects[0].Name).To(Equal("test-cluster"))
			Expect(rb.Subjects[0].Kind).To(Equal("ServiceAccount"))

			Expect(rb.RoleRef.Kind).To(Equal("Role"))
			Expect(rb.RoleRef.Name).To(Equal("test-cluster-barman-cloud"))
		})
	})

	Context("when the RoleBinding exists with matching state", func() {
		var patchCount *int

		BeforeEach(func() {
			fakeClient, patchCount = newPatchCountingClient()
			Expect(rbac.EnsureRoleBinding(ctx, fakeClient, cluster)).To(Succeed())
			*patchCount = 0
		})

		It("should not patch the RoleBinding", func() {
			err := rbac.EnsureRoleBinding(ctx, fakeClient, cluster)
			Expect(err).NotTo(HaveOccurred())
			Expect(*patchCount).To(BeZero())
		})
	})

	Context("when the RoleBinding exists with extra user-added subjects", func() {
		BeforeEach(func() {
			fakeClient = fake.NewClientBuilder().WithScheme(newScheme()).Build()
			existing := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-barman-cloud",
					Namespace: "default",
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:     "ServiceAccount",
						Name:     "user-debug-sa",
						APIGroup: "",
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "test-cluster-barman-cloud",
				},
			}
			Expect(fakeClient.Create(ctx, existing)).To(Succeed())
		})

		It("should add the plugin's subject without removing user-added ones", func() {
			err := rbac.EnsureRoleBinding(ctx, fakeClient, cluster)
			Expect(err).NotTo(HaveOccurred())

			var rb rbacv1.RoleBinding
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "test-cluster-barman-cloud",
			}, &rb)).To(Succeed())

			// Additive policy: the user-added subject must remain.
			Expect(rb.Subjects).To(ContainElement(rbacv1.Subject{
				Kind:     "ServiceAccount",
				Name:     "user-debug-sa",
				APIGroup: "",
			}))
			// The plugin's required subject must be present.
			Expect(rb.Subjects).To(ContainElement(rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      "test-cluster",
				Namespace: "default",
				APIGroup:  "",
			}))
		})
	})

	Context("when the RoleBinding exists with a stale label value", func() {
		BeforeEach(func() {
			fakeClient = fake.NewClientBuilder().WithScheme(newScheme()).Build()
			existing := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-barman-cloud",
					Namespace: "default",
					Labels: map[string]string{
						utils.KubernetesAppVersionLabelName: "0.0.0-stale",
					},
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      "test-cluster",
						Namespace: "default",
						APIGroup:  "",
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "test-cluster-barman-cloud",
				},
			}
			Expect(fakeClient.Create(ctx, existing)).To(Succeed())
		})

		It("should overwrite the stale value with the current plugin's value", func() {
			err := rbac.EnsureRoleBinding(ctx, fakeClient, cluster)
			Expect(err).NotTo(HaveOccurred())

			var rb rbacv1.RoleBinding
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "test-cluster-barman-cloud",
			}, &rb)).To(Succeed())

			Expect(rb.Labels).To(HaveKeyWithValue(
				utils.KubernetesAppVersionLabelName, metadata.Data.Version))
		})
	})

	Context("when the RoleBinding has pre-existing unrelated labels", func() {
		BeforeEach(func() {
			fakeClient = fake.NewClientBuilder().WithScheme(newScheme()).Build()
			existing := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-barman-cloud",
					Namespace: "default",
					Labels: map[string]string{
						"custom-label": "custom-value",
					},
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      "test-cluster",
						Namespace: "default",
						APIGroup:  "",
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "test-cluster-barman-cloud",
				},
			}
			Expect(fakeClient.Create(ctx, existing)).To(Succeed())
		})

		It("should preserve unrelated labels while adding the required labels", func() {
			err := rbac.EnsureRoleBinding(ctx, fakeClient, cluster)
			Expect(err).NotTo(HaveOccurred())

			var rb rbacv1.RoleBinding
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "test-cluster-barman-cloud",
			}, &rb)).To(Succeed())

			Expect(rb.Labels).To(HaveKeyWithValue("custom-label", "custom-value"))
			expectRequiredLabels(rb.Labels, "test-cluster")
		})
	})

	Context("when the RoleBinding has a divergent RoleRef", func() {
		BeforeEach(func() {
			fakeClient = fake.NewClientBuilder().WithScheme(newScheme()).Build()
			existing := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-barman-cloud",
					Namespace: "default",
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      "test-cluster",
						Namespace: "default",
						APIGroup:  "",
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "wrong-role",
				},
			}
			Expect(fakeClient.Create(ctx, existing)).To(Succeed())
		})

		It("should return a descriptive error since RoleRef is immutable", func() {
			err := rbac.EnsureRoleBinding(ctx, fakeClient, cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("RoleRef"))
			Expect(err.Error()).To(ContainSubstring("wrong-role"))
		})
	})

	Context("when an AlreadyExists race happens during a stale-cache create (plugin pod startup)", func() {
		var preExisting *rbacv1.RoleBinding

		BeforeEach(func() {
			preExisting = &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-barman-cloud",
					Namespace: "default",
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      "test-cluster",
						Namespace: "default",
						APIGroup:  "",
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "test-cluster-barman-cloud",
				},
			}

			// First Get returns NotFound (simulates cold informer
			// cache after plugin pod restart). Subsequent Gets
			// fall through to real fake-client behavior.
			gets := 0
			fakeClient = fake.NewClientBuilder().
				WithScheme(newScheme()).
				WithObjects(preExisting).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(
						ctx context.Context,
						c client.WithWatch,
						key client.ObjectKey,
						obj client.Object,
						opts ...client.GetOption,
					) error {
						gets++
						if gets == 1 {
							return apierrs.NewNotFound(
								rbacv1.Resource("rolebindings"), key.Name)
						}
						return c.Get(ctx, key, obj, opts...)
					},
				}).
				Build()
		})

		It("should tolerate the AlreadyExists and reconcile from the existing object", func() {
			err := rbac.EnsureRoleBinding(ctx, fakeClient, cluster)
			Expect(err).NotTo(HaveOccurred())

			var rb rbacv1.RoleBinding
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "test-cluster-barman-cloud",
			}, &rb)).To(Succeed())

			Expect(rb.Subjects).To(ContainElement(rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      "test-cluster",
				Namespace: "default",
				APIGroup:  "",
			}))
		})
	})

	Context("when the RoleBinding exists without labels (upgrade scenario)", func() {
		BeforeEach(func() {
			fakeClient = fake.NewClientBuilder().WithScheme(newScheme()).Build()
			existing := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-barman-cloud",
					Namespace: "default",
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      "test-cluster",
						Namespace: "default",
						APIGroup:  "",
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "test-cluster-barman-cloud",
				},
			}
			Expect(fakeClient.Create(ctx, existing)).To(Succeed())
		})

		It("should add the required labels", func() {
			err := rbac.EnsureRoleBinding(ctx, fakeClient, cluster)
			Expect(err).NotTo(HaveOccurred())

			var rb rbacv1.RoleBinding
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "test-cluster-barman-cloud",
			}, &rb)).To(Succeed())

			expectRequiredLabels(rb.Labels, "test-cluster")
		})
	})
})

var _ = Describe("EnsureRoleRules", func() {
	var (
		ctx        context.Context
		fakeClient client.Client
		objects    []barmancloudv1.ObjectStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		objects = []barmancloudv1.ObjectStore{
			newObjectStore("my-store", "default", "aws-creds"),
		}
	})

	Context("when the Role exists", func() {
		BeforeEach(func() {
			fakeClient = fake.NewClientBuilder().WithScheme(newScheme()).Build()

			// Seed a labeled Role with old rules
			cluster := newCluster("test-cluster", "default")
			oldObjects := []barmancloudv1.ObjectStore{
				newObjectStore("my-store", "default", "old-secret"),
			}
			Expect(rbac.EnsureRole(ctx, fakeClient, cluster, oldObjects)).To(Succeed())
		})

		It("should patch the rules", func() {
			roleKey := client.ObjectKey{
				Namespace: "default",
				Name:      "test-cluster-barman-cloud",
			}
			err := rbac.EnsureRoleRules(ctx, fakeClient, roleKey, objects)
			Expect(err).NotTo(HaveOccurred())

			var role rbacv1.Role
			Expect(fakeClient.Get(ctx, roleKey, &role)).To(Succeed())

			secretsRule := role.Rules[2]
			Expect(secretsRule.ResourceNames).To(ContainElement("aws-creds"))
			Expect(secretsRule.ResourceNames).NotTo(ContainElement("old-secret"))
		})

		It("should not patch when rules already match", func() {
			// Replace the seeded client with a counting one,
			// then re-seed via EnsureRole so the desired rules
			// are already in place when EnsureRoleRules runs.
			var patchCount *int
			fakeClient, patchCount = newPatchCountingClient()
			cluster := newCluster("test-cluster", "default")
			Expect(rbac.EnsureRole(ctx, fakeClient, cluster, objects)).To(Succeed())
			*patchCount = 0

			roleKey := client.ObjectKey{
				Namespace: "default",
				Name:      "test-cluster-barman-cloud",
			}
			Expect(rbac.EnsureRoleRules(ctx, fakeClient, roleKey, objects)).To(Succeed())
			Expect(*patchCount).To(BeZero())
		})

		It("should not modify labels", func() {
			roleKey := client.ObjectKey{
				Namespace: "default",
				Name:      "test-cluster-barman-cloud",
			}

			var before rbacv1.Role
			Expect(fakeClient.Get(ctx, roleKey, &before)).To(Succeed())

			Expect(rbac.EnsureRoleRules(ctx, fakeClient, roleKey, objects)).To(Succeed())

			var after rbacv1.Role
			Expect(fakeClient.Get(ctx, roleKey, &after)).To(Succeed())
			Expect(after.Labels).To(Equal(before.Labels))
		})
	})

	Context("when the Role does not exist", func() {
		BeforeEach(func() {
			fakeClient = fake.NewClientBuilder().WithScheme(newScheme()).Build()
		})

		It("should return nil", func() {
			roleKey := client.ObjectKey{
				Namespace: "default",
				Name:      "nonexistent-barman-cloud",
			}
			err := rbac.EnsureRoleRules(ctx, fakeClient, roleKey, objects)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
