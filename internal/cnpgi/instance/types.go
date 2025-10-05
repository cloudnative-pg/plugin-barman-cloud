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

package instance

import (
	"strconv"

	"k8s.io/apimachinery/pkg/types"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
)

type backupResultMetadata struct {
	timeline    string
	version     string
	name        string
	displayName string
	clusterUID  string
	pluginName  string
}

func (b backupResultMetadata) toMap() map[string]string {
	return map[string]string{
		"timeline":    b.timeline,
		"version":     b.version,
		"name":        b.name,
		"displayName": b.displayName,
		"clusterUID":  b.clusterUID,
		"pluginName":  b.pluginName,
	}
}

func newBackupResultMetadata(clusterUID types.UID, timeline int) backupResultMetadata {
	return backupResultMetadata{
		timeline:   strconv.Itoa(timeline),
		clusterUID: string(clusterUID),
		// static values
		version:     metadata.Data.Version,
		name:        metadata.Data.Name,
		displayName: metadata.Data.DisplayName,
		pluginName:  metadata.PluginName,
	}
}

func newBackupResultMetadataFromMap(m map[string]string) backupResultMetadata {
	if m == nil {
		return backupResultMetadata{}
	}

	return backupResultMetadata{
		timeline:    m["timeline"],
		version:     m["version"],
		name:        m["name"],
		displayName: m["displayName"],
		clusterUID:  m["clusterUID"],
		pluginName:  m["pluginName"],
	}
}
