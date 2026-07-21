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

package config

import (
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
)

var _ = Describe("PluginConfiguration.Validate", func() {
	It("fails when no barman object name is set", func() {
		cfg := &PluginConfiguration{}
		Expect(cfg.Validate()).To(HaveOccurred())
	})

	It("passes when only BarmanObjectName is set (backup/archive)", func() {
		cfg := &PluginConfiguration{BarmanObjectName: "my-store"}
		Expect(cfg.Validate()).To(Succeed())
	})

	It("passes when only RecoveryBarmanObjectName is set (recovery bootstrap)", func() {
		cfg := &PluginConfiguration{RecoveryBarmanObjectName: "my-store"}
		Expect(cfg.Validate()).To(Succeed())
	})

	It("passes when only ReplicaSourceBarmanObjectName is set (pg_basebackup replica cluster)", func() {
		cfg := &PluginConfiguration{ReplicaSourceBarmanObjectName: "my-store"}
		Expect(cfg.Validate()).To(Succeed())
	})
})

var _ = Describe("NewFromCluster", func() {
	enabled := true

	It("derives the replica source object store for a pg_basebackup replica cluster", func() {
		cluster := &cnpgv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster-replica", Namespace: "test-ns"},
			Spec: cnpgv1.ClusterSpec{
				Bootstrap: &cnpgv1.BootstrapConfiguration{
					PgBaseBackup: &cnpgv1.BootstrapPgBaseBackup{Source: "source"},
				},
				ReplicaCluster: &cnpgv1.ReplicaClusterConfiguration{
					Enabled: &enabled,
					Source:  "source",
				},
				ExternalClusters: []cnpgv1.ExternalCluster{
					{
						Name: "source",
						PluginConfiguration: &cnpgv1.PluginConfiguration{
							Name: metadata.PluginName,
							Parameters: map[string]string{
								"barmanObjectName": "minio-store",
								"serverName":       "cluster-example",
							},
						},
					},
				},
			},
		}

		cfg := NewFromCluster(cluster)

		// The replica source object store is derived from the external cluster plugin,
		// while the backup/archive and recovery object stores remain empty: this is the
		// distinguishing trait of a pg_basebackup replica cluster (a recovery-bootstrapped
		// replica would also populate RecoveryBarmanObjectName).
		Expect(cfg.ReplicaSourceBarmanObjectName).To(Equal("minio-store"))
		Expect(cfg.ReplicaSourceServerName).To(Equal("cluster-example"))
		Expect(cfg.BarmanObjectName).To(BeEmpty())
		Expect(cfg.RecoveryBarmanObjectName).To(BeEmpty())

		// Validate must accept it, otherwise the lifecycle hook skips sidecar injection.
		Expect(cfg.Validate()).To(Succeed())
	})

	It("ignores a replica source backed by a different plugin", func() {
		cluster := &cnpgv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster-replica", Namespace: "test-ns"},
			Spec: cnpgv1.ClusterSpec{
				Bootstrap: &cnpgv1.BootstrapConfiguration{
					PgBaseBackup: &cnpgv1.BootstrapPgBaseBackup{Source: "source"},
				},
				ReplicaCluster: &cnpgv1.ReplicaClusterConfiguration{
					Enabled: &enabled,
					Source:  "source",
				},
				ExternalClusters: []cnpgv1.ExternalCluster{
					{
						Name: "source",
						PluginConfiguration: &cnpgv1.PluginConfiguration{
							Name:       "some-other-plugin.cloudnative-pg.io",
							Parameters: map[string]string{"barmanObjectName": "minio-store"},
						},
					},
				},
			},
		}

		cfg := NewFromCluster(cluster)

		Expect(cfg.ReplicaSourceBarmanObjectName).To(BeEmpty())
		Expect(cfg.Validate()).NotTo(Succeed())
	})
})
