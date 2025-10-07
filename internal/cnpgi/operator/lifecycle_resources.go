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

	corev1 "k8s.io/api/core/v1"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"
)

func (impl LifecycleImplementation) collectSidecarResourcesForRecoveryJob(
	ctx context.Context,
	configuration *config.PluginConfiguration,
) (corev1.ResourceRequirements, error) {
	if len(configuration.RecoveryBarmanObjectName) > 0 {
		var barmanObjectStore barmancloudv1.ObjectStore
		if err := impl.Client.Get(ctx, configuration.GetRecoveryBarmanObjectKey(), &barmanObjectStore); err != nil {
			return corev1.ResourceRequirements{}, err
		}

		return barmanObjectStore.Spec.InstanceSidecarConfiguration.Resources, nil
	}

	return corev1.ResourceRequirements{}, nil
}

func (impl LifecycleImplementation) collectSidecarResourcesForPod(
	ctx context.Context,
	configuration *config.PluginConfiguration,
) (corev1.ResourceRequirements, error) {
	if len(configuration.BarmanObjectName) > 0 {
		// On a replica cluster that also archives, the designated primary
		// will use both the replica source object store and the object store
		// of the cluster.
		// In this case, we use the cluster object store for configuring
		// the resources of the sidecar container.

		var barmanObjectStore barmancloudv1.ObjectStore
		if err := impl.Client.Get(ctx, configuration.GetBarmanObjectKey(), &barmanObjectStore); err != nil {
			return corev1.ResourceRequirements{}, err
		}

		return barmanObjectStore.Spec.InstanceSidecarConfiguration.Resources, nil
	}

	if len(configuration.RecoveryBarmanObjectName) > 0 {
		// On a replica cluster that doesn't archive, the designated primary
		// uses only the replica source object store.
		// In this case, we use the replica source object store for configuring
		// the resources of the sidecar container.
		var barmanObjectStore barmancloudv1.ObjectStore
		if err := impl.Client.Get(ctx, configuration.GetRecoveryBarmanObjectKey(), &barmanObjectStore); err != nil {
			return corev1.ResourceRequirements{}, err
		}

		return barmanObjectStore.Spec.InstanceSidecarConfiguration.Resources, nil
	}

	return corev1.ResourceRequirements{}, nil
}
