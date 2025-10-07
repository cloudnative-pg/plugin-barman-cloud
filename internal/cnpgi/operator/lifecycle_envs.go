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
	"k8s.io/apimachinery/pkg/types"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"
)

func (impl LifecycleImplementation) collectAdditionalEnvs(
	ctx context.Context,
	namespace string,
	pluginConfiguration *config.PluginConfiguration,
) ([]corev1.EnvVar, error) {
	var result []corev1.EnvVar

	// TODO: check if the environment variables are clashing and in
	// that case raise an error

	if len(pluginConfiguration.BarmanObjectName) > 0 {
		envs, err := impl.collectObjectStoreEnvs(
			ctx,
			types.NamespacedName{
				Name:      pluginConfiguration.BarmanObjectName,
				Namespace: namespace,
			},
		)
		if err != nil {
			return nil, err
		}
		result = append(result, envs...)
	}

	if len(pluginConfiguration.RecoveryBarmanObjectName) > 0 {
		envs, err := impl.collectObjectStoreEnvs(
			ctx,
			types.NamespacedName{
				Name:      pluginConfiguration.RecoveryBarmanObjectName,
				Namespace: namespace,
			},
		)
		if err != nil {
			return nil, err
		}
		result = append(result, envs...)
	}

	if len(pluginConfiguration.ReplicaSourceBarmanObjectName) > 0 {
		envs, err := impl.collectObjectStoreEnvs(
			ctx,
			types.NamespacedName{
				Name:      pluginConfiguration.ReplicaSourceBarmanObjectName,
				Namespace: namespace,
			},
		)
		if err != nil {
			return nil, err
		}
		result = append(result, envs...)
	}

	return result, nil
}

func (impl LifecycleImplementation) collectObjectStoreEnvs(
	ctx context.Context,
	barmanObjectKey types.NamespacedName,
) ([]corev1.EnvVar, error) {
	var objectStore barmancloudv1.ObjectStore
	if err := impl.Client.Get(ctx, barmanObjectKey, &objectStore); err != nil {
		return nil, err
	}

	return objectStore.Spec.InstanceSidecarConfiguration.Env, nil
}
