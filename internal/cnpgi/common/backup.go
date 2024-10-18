package common

import (
	"encoding/json"
	"fmt"

	barmanapi "github.com/cloudnative-pg/barman-cloud/pkg/api"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
)

func GetCredentialsFromBackup(backup *cnpgv1.Backup) (barmanapi.BarmanCredentials, error) {
	rawCred := backup.Status.PluginMetadata["credentials"]

	var creds barmanapi.BarmanCredentials
	if err := json.Unmarshal([]byte(rawCred), &creds); err != nil {
		return barmanapi.BarmanCredentials{}, fmt.Errorf("while unmarshaling credentials: %w", err)
	}

	return creds, nil
}
