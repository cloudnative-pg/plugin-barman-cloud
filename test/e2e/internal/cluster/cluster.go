/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	v1 "github.com/cloudnative-pg/api/pkg/api/v1"
)

// TODO: improve this with what we already have in CloudNativePG e2e.
func IsReady(cluster v1.Cluster) bool {
	if cluster.Status.ReadyInstances != cluster.Spec.Instances {
		return false
	}
	for _, condition := range cluster.Status.Conditions {
		if condition.Type == string(v1.ConditionClusterReady) {
			return string(condition.Status) == string(v1.ConditionTrue)
		}
	}

	return false
}
