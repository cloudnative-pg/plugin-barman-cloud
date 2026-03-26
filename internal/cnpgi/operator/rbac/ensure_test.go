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

	barmanapi "github.com/cloudnative-pg/barman-cloud/pkg/api"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	machineryapi "github.com/cloudnative-pg/machinery/pkg/api"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/rbac"
)

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = rbacv1.AddToScheme(s)
	_ = cnpgv1.AddToScheme(s)
	_ = barmancloudv1.AddToScheme(s)
	return s
}

func newCluster(name, namespace string) *cnpgv1.Cluster {
	return &cnpgv1.Cluster{
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

		It("should create the Role with owner reference", func() {
			err := rbac.EnsureRole(ctx, fakeClient, cluster, objects)
			Expect(err).NotTo(HaveOccurred())

			var role rbacv1.Role
			err = fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "test-cluster-barman-cloud",
			}, &role)
			Expect(err).NotTo(HaveOccurred())
			Expect(role.Rules).To(HaveLen(3))

			// Verify owner reference is set to the Cluster
			Expect(role.OwnerReferences).To(HaveLen(1))
			Expect(role.OwnerReferences[0].Name).To(Equal("test-cluster"))
			Expect(role.OwnerReferences[0].Kind).To(Equal("Cluster"))
		})
	})

	Context("when the Role exists with matching rules", func() {
		BeforeEach(func() {
			fakeClient = fake.NewClientBuilder().WithScheme(newScheme()).Build()
			Expect(rbac.EnsureRole(ctx, fakeClient, cluster, objects)).To(Succeed())
		})

		It("should not patch the Role", func() {
			var before rbacv1.Role
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "test-cluster-barman-cloud",
			}, &before)).To(Succeed())

			err := rbac.EnsureRole(ctx, fakeClient, cluster, objects)
			Expect(err).NotTo(HaveOccurred())

			var after rbacv1.Role
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Namespace: "default",
				Name:      "test-cluster-barman-cloud",
			}, &after)).To(Succeed())

			Expect(after.ResourceVersion).To(Equal(before.ResourceVersion))
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

			// Owner reference must survive the patch
			Expect(role.OwnerReferences).To(HaveLen(1))
			Expect(role.OwnerReferences[0].Name).To(Equal("test-cluster"))
		})
	})
})
