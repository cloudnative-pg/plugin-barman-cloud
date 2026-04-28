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
	"github.com/cloudnative-pg/cloudnative-pg/pkg/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
)

var _ = Describe("BuildLabels", func() {
	It("should return all expected labels for a cluster without an image", func() {
		cluster := &cnpgv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-cluster",
				Namespace: "default",
			},
		}
		labels := BuildLabels(cluster)

		Expect(labels).To(HaveKeyWithValue(metadata.ClusterLabelName, "my-cluster"))
		Expect(labels).To(HaveKeyWithValue(utils.KubernetesAppLabelName, utils.AppName))
		Expect(labels).To(HaveKeyWithValue(utils.KubernetesAppInstanceLabelName, "my-cluster"))
		Expect(labels).To(HaveKeyWithValue(utils.KubernetesAppManagedByLabelName, "plugin-barman-cloud"))
		Expect(labels).To(HaveKeyWithValue(utils.KubernetesAppComponentLabelName, utils.DatabaseComponentName))
		Expect(labels).To(HaveLen(6))
	})

	It("should use the major version from the image catalog ref", func() {
		cluster := &cnpgv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pg16-cluster",
				Namespace: "default",
			},
			Spec: cnpgv1.ClusterSpec{
				ImageCatalogRef: &cnpgv1.ImageCatalogRef{
					Major: 16,
				},
			},
		}
		labels := BuildLabels(cluster)

		Expect(labels).To(HaveKeyWithValue(utils.KubernetesAppVersionLabelName, "16"))
		Expect(labels).To(HaveKeyWithValue(utils.KubernetesAppInstanceLabelName, "pg16-cluster"))
	})
})
