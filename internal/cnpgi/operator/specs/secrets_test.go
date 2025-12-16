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
	machineryapi "github.com/cloudnative-pg/machinery/pkg/api"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CollectSecretNamesFromCredentials", func() {
	Context("when collecting secrets from AWS credentials", func() {
		It("should return secret names from S3 credentials", func() {
			credentials := &barmanapi.BarmanCredentials{
				AWS: &barmanapi.S3Credentials{
					AccessKeyIDReference: &machineryapi.SecretKeySelector{
						LocalObjectReference: machineryapi.LocalObjectReference{
							Name: "aws-secret",
						},
						Key: "access-key-id",
					},
					SecretAccessKeyReference: &machineryapi.SecretKeySelector{
						LocalObjectReference: machineryapi.LocalObjectReference{
							Name: "aws-secret",
						},
						Key: "secret-access-key",
					},
				},
			}

			secrets := CollectSecretNamesFromCredentials(credentials)
			Expect(secrets).To(ContainElement("aws-secret"))
		})

		It("should handle nil AWS credentials", func() {
			credentials := &barmanapi.BarmanCredentials{}

			secrets := CollectSecretNamesFromCredentials(credentials)
			Expect(secrets).To(BeEmpty())
		})
	})

	Context("when collecting secrets from Azure credentials", func() {
		It("should return secret names when using explicit credentials", func() {
			credentials := &barmanapi.BarmanCredentials{
				Azure: &barmanapi.AzureCredentials{
					ConnectionString: &machineryapi.SecretKeySelector{
						LocalObjectReference: machineryapi.LocalObjectReference{
							Name: "azure-secret",
						},
						Key: "connection-string",
					},
				},
			}

			secrets := CollectSecretNamesFromCredentials(credentials)
			Expect(secrets).To(ContainElement("azure-secret"))
		})

		It("should return empty list when using UseDefaultAzureCredentials", func() {
			credentials := &barmanapi.BarmanCredentials{
				Azure: &barmanapi.AzureCredentials{
					UseDefaultAzureCredentials: true,
					ConnectionString: &machineryapi.SecretKeySelector{
						LocalObjectReference: machineryapi.LocalObjectReference{
							Name: "azure-secret",
						},
						Key: "connection-string",
					},
				},
			}

			secrets := CollectSecretNamesFromCredentials(credentials)
			Expect(secrets).To(BeEmpty())
		})

		It("should return empty list when using InheritFromAzureAD", func() {
			credentials := &barmanapi.BarmanCredentials{
				Azure: &barmanapi.AzureCredentials{
					InheritFromAzureAD: true,
				},
			}

			secrets := CollectSecretNamesFromCredentials(credentials)
			Expect(secrets).To(BeEmpty())
		})

		It("should return secret names for storage account and key", func() {
			credentials := &barmanapi.BarmanCredentials{
				Azure: &barmanapi.AzureCredentials{
					StorageAccount: &machineryapi.SecretKeySelector{
						LocalObjectReference: machineryapi.LocalObjectReference{
							Name: "azure-storage",
						},
						Key: "account-name",
					},
					StorageKey: &machineryapi.SecretKeySelector{
						LocalObjectReference: machineryapi.LocalObjectReference{
							Name: "azure-storage",
						},
						Key: "account-key",
					},
				},
			}

			secrets := CollectSecretNamesFromCredentials(credentials)
			Expect(secrets).To(ContainElement("azure-storage"))
		})
	})

	Context("when collecting secrets from Google credentials", func() {
		It("should return secret names from Google credentials", func() {
			credentials := &barmanapi.BarmanCredentials{
				Google: &barmanapi.GoogleCredentials{
					ApplicationCredentials: &machineryapi.SecretKeySelector{
						LocalObjectReference: machineryapi.LocalObjectReference{
							Name: "google-secret",
						},
						Key: "credentials.json",
					},
				},
			}

			secrets := CollectSecretNamesFromCredentials(credentials)
			Expect(secrets).To(ContainElement("google-secret"))
		})
	})

	Context("when collecting secrets from multiple cloud providers", func() {
		It("should return secret names from all providers", func() {
			credentials := &barmanapi.BarmanCredentials{
				AWS: &barmanapi.S3Credentials{
					AccessKeyIDReference: &machineryapi.SecretKeySelector{
						LocalObjectReference: machineryapi.LocalObjectReference{
							Name: "aws-secret",
						},
						Key: "access-key-id",
					},
				},
				Azure: &barmanapi.AzureCredentials{
					ConnectionString: &machineryapi.SecretKeySelector{
						LocalObjectReference: machineryapi.LocalObjectReference{
							Name: "azure-secret",
						},
						Key: "connection-string",
					},
				},
				Google: &barmanapi.GoogleCredentials{
					ApplicationCredentials: &machineryapi.SecretKeySelector{
						LocalObjectReference: machineryapi.LocalObjectReference{
							Name: "google-secret",
						},
						Key: "credentials.json",
					},
				},
			}

			secrets := CollectSecretNamesFromCredentials(credentials)
			Expect(secrets).To(ContainElements("aws-secret", "azure-secret", "google-secret"))
		})

		It("should skip Azure secrets when using UseDefaultAzureCredentials with other providers", func() {
			credentials := &barmanapi.BarmanCredentials{
				AWS: &barmanapi.S3Credentials{
					AccessKeyIDReference: &machineryapi.SecretKeySelector{
						LocalObjectReference: machineryapi.LocalObjectReference{
							Name: "aws-secret",
						},
						Key: "access-key-id",
					},
				},
				Azure: &barmanapi.AzureCredentials{
					UseDefaultAzureCredentials: true,
					ConnectionString: &machineryapi.SecretKeySelector{
						LocalObjectReference: machineryapi.LocalObjectReference{
							Name: "azure-secret",
						},
						Key: "connection-string",
					},
				},
			}

			secrets := CollectSecretNamesFromCredentials(credentials)
			Expect(secrets).To(ContainElement("aws-secret"))
			Expect(secrets).NotTo(ContainElement("azure-secret"))
		})
	})

	Context("when handling nil references", func() {
		It("should skip nil secret references", func() {
			credentials := &barmanapi.BarmanCredentials{
				AWS: &barmanapi.S3Credentials{
					AccessKeyIDReference: &machineryapi.SecretKeySelector{
						LocalObjectReference: machineryapi.LocalObjectReference{
							Name: "aws-secret",
						},
						Key: "access-key-id",
					},
					SecretAccessKeyReference: nil,
				},
			}

			secrets := CollectSecretNamesFromCredentials(credentials)
			Expect(secrets).To(ContainElement("aws-secret"))
			Expect(len(secrets)).To(Equal(1))
		})
	})
})
