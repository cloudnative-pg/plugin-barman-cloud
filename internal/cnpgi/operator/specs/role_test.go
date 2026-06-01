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

package specs

import (
	barmanapi "github.com/cloudnative-pg/barman-cloud/pkg/api"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	machineryapi "github.com/cloudnative-pg/machinery/pkg/api"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
)

func newTestObjectStore(name, secretName string) barmancloudv1.ObjectStore {
	return barmancloudv1.ObjectStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
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

var _ = Describe("BuildRoleRules", func() {
	It("should produce 3 rules with correct ResourceNames", func() {
		objects := []barmancloudv1.ObjectStore{
			newTestObjectStore("store-a", "secret-a"),
			newTestObjectStore("store-b", "secret-b"),
		}
		rules := BuildRoleRules(objects)
		Expect(rules).To(HaveLen(3))

		Expect(rules[0].APIGroups).To(Equal([]string{barmancloudv1.GroupVersion.Group}))
		Expect(rules[0].Resources).To(Equal([]string{"objectstores"}))
		Expect(rules[0].ResourceNames).To(ConsistOf("store-a", "store-b"))

		Expect(rules[1].APIGroups).To(Equal([]string{barmancloudv1.GroupVersion.Group}))
		Expect(rules[1].Resources).To(Equal([]string{"objectstores/status"}))
		Expect(rules[1].ResourceNames).To(ConsistOf("store-a", "store-b"))

		Expect(rules[2].APIGroups).To(Equal([]string{""}))
		Expect(rules[2].Resources).To(Equal([]string{"secrets"}))
		Expect(rules[2].ResourceNames).To(ConsistOf("secret-a", "secret-b"))
	})

	It("should produce rules with empty ResourceNames for empty input", func() {
		rules := BuildRoleRules(nil)
		Expect(rules).To(HaveLen(3))
		Expect(rules[0].ResourceNames).To(BeEmpty())
		Expect(rules[0].ResourceNames).NotTo(BeNil())
		Expect(rules[1].ResourceNames).To(BeEmpty())
		Expect(rules[2].ResourceNames).To(BeEmpty())
	})

	It("should deduplicate secret names across ObjectStores", func() {
		objects := []barmancloudv1.ObjectStore{
			newTestObjectStore("store-a", "shared-secret"),
			newTestObjectStore("store-b", "shared-secret"),
		}
		rules := BuildRoleRules(objects)
		Expect(rules[2].ResourceNames).To(Equal([]string{"shared-secret"}))
	})
})

var _ = Describe("BuildRole", func() {
	It("should set the cluster label", func() {
		cluster := &cnpgv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-cluster",
				Namespace: "default",
			},
		}
		role := BuildRole(cluster, nil)
		Expect(role.Labels).To(HaveKeyWithValue(metadata.ClusterLabelName, "my-cluster"))
		Expect(role.Name).To(Equal("my-cluster-barman-cloud"))
		Expect(role.Namespace).To(Equal("default"))
	})
})

var _ = Describe("BuildRoleRules / ObjectStoreNamesFromRole round-trip", func() {
	It("should recover the same ObjectStore names from built rules", func() {
		objects := []barmancloudv1.ObjectStore{
			newTestObjectStore("store-a", "secret-a"),
			newTestObjectStore("store-b", "secret-b"),
		}
		rules := BuildRoleRules(objects)
		role := &rbacv1.Role{Rules: rules}
		names := ObjectStoreNamesFromRole(role)
		Expect(names).To(ConsistOf("store-a", "store-b"))
	})

	It("should recover empty names from rules built with no ObjectStores", func() {
		rules := BuildRoleRules(nil)
		role := &rbacv1.Role{Rules: rules}
		names := ObjectStoreNamesFromRole(role)
		Expect(names).To(BeEmpty())
	})
})

var _ = Describe("ObjectStoreNamesFromRole", func() {
	It("should extract ObjectStore names from a well-formed Role", func() {
		role := &rbacv1.Role{
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups:     []string{barmancloudv1.GroupVersion.Group},
					Resources:     []string{"objectstores"},
					ResourceNames: []string{"store-a", "store-b"},
				},
				{
					APIGroups:     []string{""},
					Resources:     []string{"secrets"},
					ResourceNames: []string{"secret-a"},
				},
			},
		}
		Expect(ObjectStoreNamesFromRole(role)).To(Equal([]string{"store-a", "store-b"}))
	})

	It("should return nil for a Role with no matching rule", func() {
		role := &rbacv1.Role{
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups:     []string{""},
					Resources:     []string{"secrets"},
					ResourceNames: []string{"secret-a"},
				},
			},
		}
		Expect(ObjectStoreNamesFromRole(role)).To(BeNil())
	})

	It("should return nil for a Role with empty rules", func() {
		role := &rbacv1.Role{}
		Expect(ObjectStoreNamesFromRole(role)).To(BeNil())
	})

	It("should not match a rule with a different APIGroup", func() {
		role := &rbacv1.Role{
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups:     []string{"other.io"},
					Resources:     []string{"objectstores"},
					ResourceNames: []string{"store-a"},
				},
			},
		}
		Expect(ObjectStoreNamesFromRole(role)).To(BeNil())
	})

	It("should not match a rule with multiple APIGroups", func() {
		role := &rbacv1.Role{
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups:     []string{barmancloudv1.GroupVersion.Group, "other.io"},
					Resources:     []string{"objectstores"},
					ResourceNames: []string{"store-a"},
				},
			},
		}
		Expect(ObjectStoreNamesFromRole(role)).To(BeNil())
	})

	It("should not match a rule for objectstores/status", func() {
		role := &rbacv1.Role{
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups:     []string{barmancloudv1.GroupVersion.Group},
					Resources:     []string{"objectstores/status"},
					ResourceNames: []string{"store-a"},
				},
			},
		}
		Expect(ObjectStoreNamesFromRole(role)).To(BeNil())
	})
})
