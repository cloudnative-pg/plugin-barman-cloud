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
