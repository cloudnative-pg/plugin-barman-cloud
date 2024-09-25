package operator

import "github.com/cloudnative-pg/cnpg-i/pkg/identity"

const PluginName = "operator.barman-cloud.cloudnative-pg.io"

// Data is the metadata of this plugin.
var Data = identity.GetPluginMetadataResponse{
	Name:          PluginName,
	Version:       "0.0.1",
	DisplayName:   "BarmanCloudOperator",
	ProjectUrl:    "https://github.com/cloudnative-pg/plugin-barman-cloud",
	RepositoryUrl: "https://github.com/cloudnative-pg/plugin-barman-cloud",
	License:       "APACHE 2.0",
	LicenseUrl:    "https://github.com/cloudnative-pg/plugin-barman-cloud/LICENSE",
	Maturity:      "alpha",
}
