package operator

import (
	"context"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"
)

func (impl LifecycleImplementation) collectSidecarStartupProbeForRecoveryJob(
	ctx context.Context,
	configuration *config.PluginConfiguration,
) (*barmancloudv1.ProbeConfig, error) {
	if len(configuration.RecoveryBarmanObjectName) > 0 {
		var barmanObjectStore barmancloudv1.ObjectStore
		if err := impl.Client.Get(ctx, configuration.GetRecoveryBarmanObjectKey(), &barmanObjectStore); err != nil {
			return nil, err
		}

		return barmanObjectStore.Spec.InstanceSidecarConfiguration.StartupProbe, nil
	}

	return nil, nil
}

func (impl LifecycleImplementation) collectSidecarStartupProbeForInstancePod(
	ctx context.Context,
	configuration *config.PluginConfiguration,
) (*barmancloudv1.ProbeConfig, error) {
	if len(configuration.BarmanObjectName) > 0 {
		// On a replica cluster that also archives, the designated primary
		// will use both the replica source object store and the object store
		// of the cluster.
		// In this case, we use the cluster object store for configuring
		// the startup probe of the sidecar container.

		var barmanObjectStore barmancloudv1.ObjectStore
		if err := impl.Client.Get(ctx, configuration.GetBarmanObjectKey(), &barmanObjectStore); err != nil {
			return nil, err
		}

		return barmanObjectStore.Spec.InstanceSidecarConfiguration.StartupProbe, nil
	}

	if len(configuration.RecoveryBarmanObjectName) > 0 {
		// On a replica cluster that doesn't archive, the designated primary
		// uses only the replica source object store.
		// In this case, we use the replica source object store for configuring
		// the startup probe of the sidecar container.
		var barmanObjectStore barmancloudv1.ObjectStore
		if err := impl.Client.Get(ctx, configuration.GetRecoveryBarmanObjectKey(), &barmanObjectStore); err != nil {
			return nil, err
		}

		return barmanObjectStore.Spec.InstanceSidecarConfiguration.StartupProbe, nil
	}

	return nil, nil
}
