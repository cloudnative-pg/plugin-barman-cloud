package specs

import (
	machineryapi "github.com/cloudnative-pg/machinery/pkg/api"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
)

func collectSecretNames(object *barmancloudv1.ObjectStore) []string {
	if object == nil {
		return nil
	}

	var references []*machineryapi.SecretKeySelector
	if object.Spec.Configuration.AWS != nil {
		references = append(
			references,
			object.Spec.Configuration.AWS.AccessKeyIDReference,
			object.Spec.Configuration.AWS.SecretAccessKeyReference,
			object.Spec.Configuration.AWS.RegionReference,
			object.Spec.Configuration.AWS.SessionToken,
		)
	}
	if object.Spec.Configuration.Azure != nil {
		references = append(
			references,
			object.Spec.Configuration.Azure.ConnectionString,
			object.Spec.Configuration.Azure.StorageAccount,
			object.Spec.Configuration.Azure.StorageKey,
			object.Spec.Configuration.Azure.StorageSasToken,
		)
	}
	if object.Spec.Configuration.Google != nil {
		references = append(
			references,
			object.Spec.Configuration.Google.ApplicationCredentials,
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
