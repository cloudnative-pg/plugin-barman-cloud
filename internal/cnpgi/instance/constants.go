package instance

import "github.com/cloudnative-pg/cnpg-i/pkg/identity"

// PluginName is the name of the plugin from the instance manager
// Point-of-view
const PluginName = "instance.barman-cloud.cloudnative-pg.io"

// Data is the metadata of this plugin.
var Data = identity.GetPluginMetadataResponse{
	Name:          PluginName,
	Version:       "0.0.1",
	DisplayName:   "BarmanCloudInstance",
	ProjectUrl:    "https://github.com/cloudnative-pg/plugin-barman-cloud",
	RepositoryUrl: "https://github.com/cloudnative-pg/plugin-barman-cloud",
	License:       "APACHE 2.0",
	LicenseUrl:    "https://github.com/cloudnative-pg/plugin-barman-cloud/LICENSE",
	Maturity:      "alpha",
}
