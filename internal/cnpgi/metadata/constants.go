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

package metadata

import "github.com/cloudnative-pg/cnpg-i/pkg/identity"

// PluginName is the name of the plugin from the instance manager
// Point-of-view
const PluginName = "barman-cloud.cloudnative-pg.io"

const (
	// ClusterLabelName is the label applied to RBAC resources created
	// by this plugin. Its value is the name of the owning Cluster.
	//
	// Discovery contract: internal/controller/objectstore_controller.go
	// selects Roles by this key when an ObjectStore is reconciled.
	// Renaming or removing the label would break that controller; new
	// recommended-label keys are added alongside it, never in place
	// of it.
	ClusterLabelName = "barmancloud.cnpg.io/cluster"

	// AppLabelValue is the value applied to app.kubernetes.io/name on
	// every plugin-managed object. It identifies the application as
	// the Barman Cloud plugin (see issue #545).
	AppLabelValue = "barman-cloud-plugin"

	// ManagedByLabelValue is the value applied to app.kubernetes.io/managed-by
	// on every plugin-managed object. It identifies this plugin as
	// the controller responsible for the object.
	ManagedByLabelValue = "plugin-barman-cloud"

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
	Version:       "0.12.0", // x-release-please-version
	DisplayName:   "BarmanCloudInstance",
	ProjectUrl:    "https://github.com/cloudnative-pg/plugin-barman-cloud",
	RepositoryUrl: "https://github.com/cloudnative-pg/plugin-barman-cloud",
	License:       "APACHE 2.0",
	LicenseUrl:    "https://github.com/cloudnative-pg/plugin-barman-cloud/LICENSE",
	Maturity:      "alpha",
}
