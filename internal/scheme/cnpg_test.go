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

package scheme

import (
	"context"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ = Describe("AddCNPGToScheme", func() {
	var s *runtime.Scheme

	BeforeEach(func() {
		s = runtime.NewScheme()
	})

	AfterEach(func() {
		viper.Reset()
	})

	It("should register CNPG types under the default group and version", func() {
		AddCNPGToScheme(context.Background(), s)

		gvks, _, err := s.ObjectKinds(&cnpgv1.Cluster{})
		Expect(err).NotTo(HaveOccurred())
		Expect(gvks).To(ContainElement(schema.GroupVersionKind{
			Group:   cnpgv1.SchemeGroupVersion.Group,
			Version: cnpgv1.SchemeGroupVersion.Version,
			Kind:    "Cluster",
		}))
	})

	It("should register Backup and ScheduledBackup under the default group", func() {
		AddCNPGToScheme(context.Background(), s)

		gvks, _, err := s.ObjectKinds(&cnpgv1.Backup{})
		Expect(err).NotTo(HaveOccurred())
		Expect(gvks).To(ContainElement(HaveField("Group", cnpgv1.SchemeGroupVersion.Group)))

		gvks, _, err = s.ObjectKinds(&cnpgv1.ScheduledBackup{})
		Expect(err).NotTo(HaveOccurred())
		Expect(gvks).To(ContainElement(HaveField("Group", cnpgv1.SchemeGroupVersion.Group)))
	})

	It("should register CNPG types under a custom group", func() {
		viper.Set("custom-cnpg-group", "mycompany.io")

		AddCNPGToScheme(context.Background(), s)

		gvks, _, err := s.ObjectKinds(&cnpgv1.Cluster{})
		Expect(err).NotTo(HaveOccurred())
		Expect(gvks).To(ContainElement(schema.GroupVersionKind{
			Group:   "mycompany.io",
			Version: cnpgv1.SchemeGroupVersion.Version,
			Kind:    "Cluster",
		}))
		// The default group must not be registered
		Expect(s.Recognizes(schema.GroupVersionKind{
			Group:   cnpgv1.SchemeGroupVersion.Group,
			Version: cnpgv1.SchemeGroupVersion.Version,
			Kind:    "Cluster",
		})).To(BeFalse())
	})

	It("should register CNPG types under a custom version", func() {
		viper.Set("custom-cnpg-version", "v2")

		AddCNPGToScheme(context.Background(), s)

		gvks, _, err := s.ObjectKinds(&cnpgv1.Cluster{})
		Expect(err).NotTo(HaveOccurred())
		Expect(gvks).To(ContainElement(schema.GroupVersionKind{
			Group:   cnpgv1.SchemeGroupVersion.Group,
			Version: "v2",
			Kind:    "Cluster",
		}))
	})

	It("should register CNPG types under both a custom group and custom version", func() {
		viper.Set("custom-cnpg-group", "mycompany.io")
		viper.Set("custom-cnpg-version", "v2")

		AddCNPGToScheme(context.Background(), s)

		gvks, _, err := s.ObjectKinds(&cnpgv1.Cluster{})
		Expect(err).NotTo(HaveOccurred())
		Expect(gvks).To(ContainElement(schema.GroupVersionKind{
			Group:   "mycompany.io",
			Version: "v2",
			Kind:    "Cluster",
		}))
	})
})
