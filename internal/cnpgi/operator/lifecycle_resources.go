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
		// On a replica cluster, the designated primary will use both the
		// replica source object store and the object store of the cluster.
		// The designed primary role can change without the Pod being
		// recreated.
		// In this case, we use the cluster object store for configuring
		// the resources created by the sidecar.

		var barmanObjectStore barmancloudv1.ObjectStore
		if err := impl.Client.Get(ctx, configuration.GetBarmanObjectKey(), &barmanObjectStore); err != nil {
			return corev1.ResourceRequirements{}, err
		}

		return barmanObjectStore.Spec.InstanceSidecarConfiguration.Resources, nil
	}

	return corev1.ResourceRequirements{}, nil
}
