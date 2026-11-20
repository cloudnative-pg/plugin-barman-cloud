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

package instance

import (
	barmanapi "github.com/cloudnative-pg/barman-cloud/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("backupResultMetadata", func() {
	newObjectStore := func(endpointURL string) *barmancloudv1.ObjectStore {
		return &barmancloudv1.ObjectStore{
			ObjectMeta: metav1.ObjectMeta{Name: "store", Namespace: "default"},
			Spec: barmancloudv1.ObjectStoreSpec{
				Configuration: barmanapi.BarmanObjectStoreConfiguration{
					DestinationPath: "s3://bucket/path",
					EndpointURL:     endpointURL,
				},
			},
		}
	}

	It("records the location the backup was written to", func() {
		m := newBackupResultMetadata("uid", 2, newObjectStore("https://s3.example.com"), "mydb-20261003T120000").toMap()
		Expect(m).To(Equal(map[string]string{
			"timeline":         "2",
			"clusterUID":       "uid",
			"version":          metadata.Data.Version,
			"name":             metadata.Data.Name,
			"displayName":      metadata.Data.DisplayName,
			"pluginName":       metadata.PluginName,
			"barmanObjectName": "store",
			"serverName":       "mydb-20261003T120000",
			"destinationPath":  "s3://bucket/path",
			"endpointURL":      "https://s3.example.com",
		}))
	})

	It("omits the endpointURL when it is not set", func() {
		m := newBackupResultMetadata("uid", 1, newObjectStore(""), "mydb").toMap()
		Expect(m).NotTo(HaveKey("endpointURL"))
		Expect(m).To(HaveKeyWithValue("destinationPath", "s3://bucket/path"))
	})

	It("omits the location keys of a missing object store", func() {
		m := newBackupResultMetadata("uid", 1, nil, "mydb").toMap()
		Expect(m).To(HaveKeyWithValue("serverName", "mydb"))
		Expect(m).NotTo(HaveKey("barmanObjectName"))
		Expect(m).NotTo(HaveKey("destinationPath"))
		Expect(m).NotTo(HaveKey("endpointURL"))
	})

	It("omits the serverName when it is not set", func() {
		m := newBackupResultMetadata("uid", 1, newObjectStore(""), "").toMap()
		Expect(m).NotTo(HaveKey("serverName"))
		Expect(m).To(HaveKeyWithValue("barmanObjectName", "store"))
	})

	It("redacts credentials embedded in the endpointURL", func() {
		m := newBackupResultMetadata("uid", 1, newObjectStore("https://user:secret@s3.example.com"), "mydb").toMap()
		Expect(m).To(HaveKeyWithValue("endpointURL", "https://user:xxxxx@s3.example.com"))
	})
})

var _ = Describe("redactURL", func() {
	It("masks passwords embedded in the URL", func() {
		Expect(redactURL("https://user:secret@s3.example.com")).To(Equal("https://user:xxxxx@s3.example.com"))
	})

	It("leaves URLs without credentials untouched", func() {
		Expect(redactURL("s3://bucket/path")).To(Equal("s3://bucket/path"))
		Expect(redactURL("")).To(BeEmpty())
	})
})
