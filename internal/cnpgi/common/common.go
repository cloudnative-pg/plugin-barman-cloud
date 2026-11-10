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

	// PgWalVolumePgWalPath is the path of the pg_wal directory inside the WAL volume,
	// used when a separate WAL storage is configured. During a restore the pg_wal
	// directory is moved here and symlinked back into PGDATA.
	PgWalVolumePgWalPath = "/var/lib/postgresql/wal/pg_wal"
)

// GetRestoreCABundleEnv gets the enveronment variables to be used when custom
// Object Store CA is present
func GetRestoreCABundleEnv(configuration *barmanapi.BarmanObjectStoreConfiguration) []string {
	var env []string

	if configuration.EndpointCA != nil && configuration.AWS != nil {
		env = append(env, fmt.Sprintf("AWS_CA_BUNDLE=%s", BarmanBackupEndpointCACertificateLocation))
	} else if configuration.EndpointCA != nil && configuration.Azure != nil {
		env = append(env, fmt.Sprintf("REQUESTS_CA_BUNDLE=%s", BarmanBackupEndpointCACertificateLocation))
	}
	return env
}

// MergeEnv merges all the values inside incomingEnv into env.
func MergeEnv(env []string, incomingEnv []string) []string {
	byName := make(map[string]string, len(env)+len(incomingEnv))

	for _, item := range env {
		if name, ok := envName(item); ok {
			byName[name] = item
		}
	}

	for _, item := range incomingEnv {
		if name, ok := envName(item); ok {
			byName[name] = item
		}
	}

	result := make([]string, 0, len(byName))
	for _, item := range byName {
		result = append(result, item)
	}

	return result
}

// envName returns the name of a NAME=VALUE entry.
func envName(item string) (string, bool) {
	name, _, found := strings.Cut(item, "=")

	return name, found
}

// BuildCertificateFilePath builds the path to the barman objectStore certificate
func BuildCertificateFilePath(objectStoreName string) string {
	return path.Join(metadata.BarmanCertificatesPath, objectStoreName, metadata.BarmanCertificatesFileName)
}
