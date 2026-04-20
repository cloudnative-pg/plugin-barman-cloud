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

package specs

import (
	"fmt"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/utils"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
)

// GetRequiredLabels returns the labels that must be present on all plugin-managed objects for a given Cluster.
func GetRequiredLabels(cluster *cnpgv1.Cluster) map[string]string {
	requiredLabels := map[string]string{
		metadata.ClusterLabelName: cluster.Name,
		// Kubernetes recommended labels
		utils.KubernetesAppLabelName:          utils.AppName,
		utils.KubernetesAppInstanceLabelName:  cluster.Name,
		utils.KubernetesAppManagedByLabelName: "plugin-barman-cloud",
		utils.KubernetesAppComponentLabelName: utils.DatabaseComponentName,
	}

	if version, err := cluster.GetPostgresqlMajorVersion(); err == nil && version != 0 {
		requiredLabels[utils.KubernetesAppVersionLabelName] = fmt.Sprint(version)
	}

	return requiredLabels
}
