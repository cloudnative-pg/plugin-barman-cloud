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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
