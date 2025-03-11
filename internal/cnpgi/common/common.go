package common

import (
	"fmt"
	"path"
	"strings"

	barmanapi "github.com/cloudnative-pg/barman-cloud/pkg/api"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
)

// TODO: refactor.
const (
	// ScratchDataDirectory is the directory to be used for scratch data.
	ScratchDataDirectory = "/controller"

	// CertificatesDir location to store the certificates.
	CertificatesDir = ScratchDataDirectory + "/certificates/"

	// BarmanBackupEndpointCACertificateLocation is the location where the barman endpoint
	// CA certificate is stored.
	BarmanBackupEndpointCACertificateLocation = CertificatesDir + BarmanBackupEndpointCACertificateFileName

	// BarmanBackupEndpointCACertificateFileName is the name of the file in which the barman endpoint
	// CA certificate for backups is stored.
	BarmanBackupEndpointCACertificateFileName = "backup-" + BarmanEndpointCACertificateFileName

	// BarmanRestoreEndpointCACertificateFileName is the name of the file in which the barman endpoint
	// CA certificate for restores is stored.
	BarmanRestoreEndpointCACertificateFileName = "restore-" + BarmanEndpointCACertificateFileName

	// BarmanEndpointCACertificateFileName is the name of the file in which the barman endpoint
	// CA certificate is stored.
	BarmanEndpointCACertificateFileName = "barman-ca.crt"
)

// GetRestoreCABundleEnv gets the enveronment variables to be used when custom
// Object Store CA is present
func GetRestoreCABundleEnv(configuration *barmanapi.BarmanObjectStoreConfiguration) []string {
	var env []string

	if configuration.EndpointCA != nil && configuration.BarmanCredentials.AWS != nil {
		env = append(env, fmt.Sprintf("AWS_CA_BUNDLE=%s", BarmanBackupEndpointCACertificateLocation))
	} else if configuration.EndpointCA != nil && configuration.BarmanCredentials.Azure != nil {
		env = append(env, fmt.Sprintf("REQUESTS_CA_BUNDLE=%s", BarmanBackupEndpointCACertificateLocation))
	}
	return env
}

// MergeEnv merges all the values inside incomingEnv into env.
func MergeEnv(env []string, incomingEnv []string) []string {
	result := make([]string, len(env), len(env)+len(incomingEnv))
	copy(result, env)

	for _, incomingItem := range incomingEnv {
		incomingKV := strings.SplitAfterN(incomingItem, "=", 2)
		if len(incomingKV) != 2 {
			continue
		}

		found := false
		for idx, item := range result {
			if strings.HasPrefix(item, incomingKV[0]) {
				result[idx] = incomingItem
				found = true
			}
		}
		if !found {
			result = append(result, incomingItem)
		}
	}

	return result
}

// BuildCertificateFilePath builds the path to the barman objectStore certificate
func BuildCertificateFilePath(objectStoreName string) string {
	return path.Join(metadata.BarmanCertificatesPath, objectStoreName, metadata.BarmanCertificatesFileName)
}
