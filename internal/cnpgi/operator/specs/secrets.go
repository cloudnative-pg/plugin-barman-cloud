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
