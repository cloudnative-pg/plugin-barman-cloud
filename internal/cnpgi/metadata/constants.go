package metadata

import "github.com/cloudnative-pg/cnpg-i/pkg/identity"

// PluginName is the name of the plugin from the instance manager
// Point-of-view
const PluginName = "barman-cloud.cloudnative-pg.io"

const (
	// CheckEmptyWalArchiveFile is the name of the file in the PGDATA that,
	// if present, requires the WAL archiver to check that the backup object
	// store is empty.
	CheckEmptyWalArchiveFile = ".check-empty-wal-archive"

	// BarmanCertificatesPath is the path where the Barman
	// certificates will be installed
	BarmanCertificatesPath = "/barman-certificates"

	// BarmanCertificatesFileName is the path where the Barman
	// certificates will be used
	BarmanCertificatesFileName = "barman-ca.crt"
)

// Data is the metadata of this plugin.
var Data = identity.GetPluginMetadataResponse{
	Name:          PluginName,
	Version:       "0.2.0", // x-release-please-version
	DisplayName:   "BarmanCloudInstance",
	ProjectUrl:    "https://github.com/cloudnative-pg/plugin-barman-cloud",
	RepositoryUrl: "https://github.com/cloudnative-pg/plugin-barman-cloud",
	License:       "APACHE 2.0",
	LicenseUrl:    "https://github.com/cloudnative-pg/plugin-barman-cloud/LICENSE",
	Maturity:      "alpha",
}
