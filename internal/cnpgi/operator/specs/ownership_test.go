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
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("SetControllerReference", func() {
	It("should set the owner reference from the owner's TypeMeta", func() {
		owner := &cnpgv1.Cluster{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "postgresql.cnpg.io/v1",
				Kind:       "Cluster",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-cluster",
				Namespace: "default",
				UID:       types.UID("test-uid"),
			},
		}
		controlled := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-role",
				Namespace: "default",
			},
		}

		Expect(SetControllerReference(owner, controlled)).To(Succeed())
		Expect(controlled.OwnerReferences).To(HaveLen(1))
		Expect(controlled.OwnerReferences[0].APIVersion).To(Equal("postgresql.cnpg.io/v1"))
		Expect(controlled.OwnerReferences[0].Kind).To(Equal("Cluster"))
		Expect(controlled.OwnerReferences[0].Name).To(Equal("my-cluster"))
		Expect(controlled.OwnerReferences[0].UID).To(Equal(types.UID("test-uid")))
		Expect(*controlled.OwnerReferences[0].Controller).To(BeTrue())
		Expect(*controlled.OwnerReferences[0].BlockOwnerDeletion).To(BeTrue())
	})

	It("should work with a custom CNPG API group", func() {
		owner := &cnpgv1.Cluster{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "mycompany.io/v1",
				Kind:       "Cluster",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-cluster",
				UID:  types.UID("test-uid"),
			},
		}
		controlled := &rbacv1.Role{}

		Expect(SetControllerReference(owner, controlled)).To(Succeed())
		Expect(controlled.OwnerReferences[0].APIVersion).To(Equal("mycompany.io/v1"))
	})

	It("should fail when the owner has no GVK set", func() {
		owner := &cnpgv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-cluster",
			},
		}
		controlled := &rbacv1.Role{}

		err := SetControllerReference(owner, controlled)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("has no GVK set"))
	})

	It("should fail when the owner does not implement runtime.Object", func() {
		// metav1.ObjectMeta satisfies metav1.Object but not runtime.Object.
		owner := &metav1.ObjectMeta{Name: "my-cluster"}
		controlled := &rbacv1.Role{}

		err := SetControllerReference(owner, controlled)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("is not a runtime.Object"))
	})
})
