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
)

// CollectSecretNamesFromCredentials collects the names of the secrets
func CollectSecretNamesFromCredentials(barmanCredentials *barmanapi.BarmanCredentials) []string {
	var references []*machineryapi.SecretKeySelector
	if barmanCredentials.AWS != nil {
		references = append(
			references,
			barmanCredentials.AWS.AccessKeyIDReference,
			barmanCredentials.AWS.SecretAccessKeyReference,
			barmanCredentials.AWS.RegionReference,
			barmanCredentials.AWS.SessionToken,
		)
	}
	if barmanCredentials.Azure != nil {
		references = append(
			references,
			barmanCredentials.Azure.ConnectionString,
			barmanCredentials.Azure.StorageAccount,
			barmanCredentials.Azure.StorageKey,
			barmanCredentials.Azure.StorageSasToken,
		)
	}
	if barmanCredentials.Google != nil {
		references = append(
			references,
			barmanCredentials.Google.ApplicationCredentials,
		)
	}

	result := make([]string, 0, len(references))
	for _, reference := range references {
		if reference == nil {
			continue
		}
		result = append(result, reference.Name)
	}

	// TODO: stringset belongs to machinery :(

	return result
}
