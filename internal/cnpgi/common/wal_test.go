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

package common

import (
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"
)

var _ = Describe("resolveRestoreObjectStore", func() {
	const (
		namespace = "test-ns"
		instance  = "cluster-1"
	)

	// newConfig builds a PluginConfiguration with distinct, recognizable names
	// for every candidate object store, so each test can assert exactly which
	// one the routing selected.
	newConfig := func(currentPrimary, replicaSourceObject string) *config.PluginConfiguration {
		return &config.PluginConfiguration{
			Cluster: &cnpgv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
				Status:     cnpgv1.ClusterStatus{CurrentPrimary: currentPrimary},
			},
			BarmanObjectName:              "cluster-store",
			ServerName:                    "cluster-server",
			RecoveryBarmanObjectName:      "recovery-store",
			RecoveryServerName:            "recovery-server",
			ReplicaSourceBarmanObjectName: replicaSourceObject,
			ReplicaSourceServerName:       "replica-server",
		}
	}

	DescribeTable(
		"selects the correct object store for restoring WAL files",
		func(cfg *config.PluginConfiguration, wantServer, wantObject string) {
			gotServer, gotKey := resolveRestoreObjectStore(cfg, instance)

			Expect(gotServer).To(Equal(wantServer))
			Expect(gotKey.Name).To(Equal(wantObject))
			Expect(gotKey.Namespace).To(Equal(namespace))
		},

		// The regression this guards: during a designated-primary promotion the
		// instance is already the current primary while still in recovery, and it
		// must pull remaining WALs from the replica source. The routing decision does
		// not depend on the promotion token, so this single case covers both
		// switchover and failover.
		Entry("designated primary in promotion -> replica source",
			newConfig(instance, "replica-store"),
			"replica-server", "replica-store"),

		// Guards the len(ReplicaSourceBarmanObjectName) > 0 gate: a current primary
		// without a barman-backed replica source (plain HA primary, or a replica
		// cluster whose source is streaming-only) must use the cluster store, not
		// an empty-named replica source key.
		Entry("current primary without a replica source -> cluster store",
			newConfig(instance, ""),
			"cluster-server", "cluster-store"),

		// Bootstrap / PITR: no current primary yet. Recovery wins even if a replica
		// source happens to be configured.
		Entry("no current primary -> recovery store",
			newConfig("", "replica-store"),
			"recovery-server", "recovery-store"),

		Entry("ordinary standby -> cluster store",
			newConfig("cluster-2", ""),
			"cluster-server", "cluster-store"),

		// A non-primary instance must never route to the replica source, even when
		// one is configured: only the designated primary catches up from the source.
		Entry("standby in a replica cluster -> cluster store",
			newConfig("cluster-2", "replica-store"),
			"cluster-server", "cluster-store"),
	)
})
