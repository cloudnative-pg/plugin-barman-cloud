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

package operator

import (
	"context"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"
)

func (impl LifecycleImplementation) collectSidecarImageForRecoveryJob(
	ctx context.Context,
	configuration *config.PluginConfiguration,
) (string, error) {
	if len(configuration.RecoveryBarmanObjectName) == 0 {
		return "", nil
	}

	var objectStore barmancloudv1.ObjectStore
	if err := impl.Client.Get(ctx, configuration.GetRecoveryBarmanObjectKey(), &objectStore); err != nil {
		return "", err
	}

	return objectStore.Spec.InstanceSidecarConfiguration.SidecarImage, nil
}

func (impl LifecycleImplementation) collectSidecarImageForPod(
	ctx context.Context,
	configuration *config.PluginConfiguration,
) (string, error) {
	// Keep the same precedence used for sidecar resources and arguments.
	switch {
	case len(configuration.BarmanObjectName) > 0:
		var objectStore barmancloudv1.ObjectStore
		if err := impl.Client.Get(ctx, configuration.GetBarmanObjectKey(), &objectStore); err != nil {
			return "", err
		}
		return objectStore.Spec.InstanceSidecarConfiguration.SidecarImage, nil

	case len(configuration.RecoveryBarmanObjectName) > 0:
		var objectStore barmancloudv1.ObjectStore
		if err := impl.Client.Get(ctx, configuration.GetRecoveryBarmanObjectKey(), &objectStore); err != nil {
			return "", err
		}
		return objectStore.Spec.InstanceSidecarConfiguration.SidecarImage, nil

	case len(configuration.ReplicaSourceBarmanObjectName) > 0:
		var objectStore barmancloudv1.ObjectStore
		if err := impl.Client.Get(ctx, configuration.GetReplicaSourceBarmanObjectKey(), &objectStore); err != nil {
			return "", err
		}
		return objectStore.Spec.InstanceSidecarConfiguration.SidecarImage, nil

	default:
		return "", nil
	}
}
