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
	"path"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"
)

// barmanCertificatesVolumeName is the name of the volume that hosts
// the barman certificates to be used
const barmanCertificatesVolumeName = "barman-certificates"

func (impl LifecycleImplementation) collectAdditionalCertificates(
	ctx context.Context,
	pluginConfiguration *config.PluginConfiguration,
) ([]corev1.VolumeProjection, error) {
	var result []corev1.VolumeProjection

	for _, barmanObjectKey := range pluginConfiguration.GetReferredBarmanObjectsKey() {
		certs, err := impl.collectObjectStoreCertificates(ctx, barmanObjectKey)
		if err != nil {
			return nil, err
		}
		result = append(result, certs...)
	}

	return result, nil
}

func (impl LifecycleImplementation) collectObjectStoreCertificates(
	ctx context.Context,
	barmanObjectKey types.NamespacedName,
) ([]corev1.VolumeProjection, error) {
	var objectStore barmancloudv1.ObjectStore
	if err := impl.Client.Get(ctx, barmanObjectKey, &objectStore); err != nil {
		return nil, err
	}

	endpointCA := objectStore.Spec.Configuration.EndpointCA
	if endpointCA == nil {
		return nil, nil
	}

	return []corev1.VolumeProjection{
		{
			Secret: &corev1.SecretProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: endpointCA.Name,
				},
				Items: []corev1.KeyToPath{
					{
						Key: endpointCA.Key,
						Path: path.Join(
							barmanObjectKey.Name,
							metadata.BarmanCertificatesFileName,
						),
					},
				},
			},
		},
	}, nil
}
