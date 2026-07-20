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
	"github.com/cloudnative-pg/cnpg-i/pkg/identity"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IdentityImplementation", func() {
	Describe("GetPluginCapabilities", func() {
		It("declares the WAL, backup, metrics and restore-job services", func(ctx SpecContext) {
			impl := IdentityImplementation{}
			response, err := impl.GetPluginCapabilities(ctx, &identity.GetPluginCapabilitiesRequest{})
			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())

			var serviceTypes []identity.PluginCapability_Service_Type
			for _, capability := range response.GetCapabilities() {
				serviceTypes = append(serviceTypes, capability.GetService().GetType())
			}

			// The instance sidecar now runs the phase-0 restore in-process, so it must
			// advertise TYPE_RESTORE_JOB alongside the services it already served.
			Expect(serviceTypes).To(ConsistOf(
				identity.PluginCapability_Service_TYPE_WAL_SERVICE,
				identity.PluginCapability_Service_TYPE_BACKUP_SERVICE,
				identity.PluginCapability_Service_TYPE_METRICS,
				identity.PluginCapability_Service_TYPE_RESTORE_JOB,
			))
		})
	})
})
