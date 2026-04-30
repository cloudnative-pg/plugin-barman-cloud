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
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/utils"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
)

// BuildLabels returns the Kubernetes recommended labels applied to
// every object managed by this plugin for the given Cluster. See
// https://github.com/cloudnative-pg/plugin-barman-cloud/issues/545.
func BuildLabels(cluster *cnpgv1.Cluster) map[string]string {
	return map[string]string{
		metadata.ClusterLabelName:             cluster.Name,
		utils.KubernetesAppLabelName:          metadata.AppLabelValue,
		utils.KubernetesAppInstanceLabelName:  cluster.Name,
		utils.KubernetesAppVersionLabelName:   metadata.Data.Version,
		utils.KubernetesAppComponentLabelName: utils.DatabaseComponentName,
		utils.KubernetesAppManagedByLabelName: metadata.ManagedByLabelValue,
	}
}
