package operator

import (
	"context"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"
)

// probeAccessor is a function type that extracts a specific probe configuration from an ObjectStore
type probeAccessor func(*barmancloudv1.ObjectStore) *barmancloudv1.ProbeConfig

// collectSidecarProbeForRecoveryJob is a generic function to collect probe configurations for recovery jobs
func (impl LifecycleImplementation) collectSidecarProbeForRecoveryJob(
	ctx context.Context,
	configuration *config.PluginConfiguration,
	accessor probeAccessor,
) (*barmancloudv1.ProbeConfig, error) {
	if len(configuration.RecoveryBarmanObjectName) > 0 {
		var barmanObjectStore barmancloudv1.ObjectStore
		if err := impl.Client.Get(ctx, configuration.GetRecoveryBarmanObjectKey(), &barmanObjectStore); err != nil {
			return nil, err
		}

		return accessor(&barmanObjectStore), nil
	}

	return nil, nil
}

// collectSidecarProbeForInstancePod is a generic function to collect probe configurations for instance pods
func (impl LifecycleImplementation) collectSidecarProbeForInstancePod(
	ctx context.Context,
	configuration *config.PluginConfiguration,
	accessor probeAccessor,
	probeType string,
) (*barmancloudv1.ProbeConfig, error) {
	if len(configuration.BarmanObjectName) > 0 {
		// On a replica cluster that also archives, the designated primary
		// will use both the replica source object store and the object store
		// of the cluster.
		// In this case, we use the cluster object store for configuring
		// the probe of the sidecar container.

		var barmanObjectStore barmancloudv1.ObjectStore
		if err := impl.Client.Get(ctx, configuration.GetBarmanObjectKey(), &barmanObjectStore); err != nil {
			return nil, err
		}

		return accessor(&barmanObjectStore), nil
	}

	if len(configuration.RecoveryBarmanObjectName) > 0 {
		// On a replica cluster that doesn't archive, the designated primary
		// uses only the replica source object store.
		// In this case, we use the replica source object store for configuring
		// the probe of the sidecar container.
		var barmanObjectStore barmancloudv1.ObjectStore
		if err := impl.Client.Get(ctx, configuration.GetRecoveryBarmanObjectKey(), &barmanObjectStore); err != nil {
			return nil, err
		}

		return accessor(&barmanObjectStore), nil
	}

	return nil, nil
}

// Specific probe collection methods that use the generic functions

func (impl LifecycleImplementation) collectSidecarStartupProbeForRecoveryJob(
	ctx context.Context,
	configuration *config.PluginConfiguration,
) (*barmancloudv1.ProbeConfig, error) {
	return impl.collectSidecarProbeForRecoveryJob(ctx, configuration, func(store *barmancloudv1.ObjectStore) *barmancloudv1.ProbeConfig {
		return store.Spec.InstanceSidecarConfiguration.StartupProbe
	})
}

func (impl LifecycleImplementation) collectSidecarStartupProbeForInstancePod(
	ctx context.Context,
	configuration *config.PluginConfiguration,
) (*barmancloudv1.ProbeConfig, error) {
	return impl.collectSidecarProbeForInstancePod(ctx, configuration, func(store *barmancloudv1.ObjectStore) *barmancloudv1.ProbeConfig {
		return store.Spec.InstanceSidecarConfiguration.StartupProbe
	}, "startup")
}

func (impl LifecycleImplementation) collectSidecarLivenessProbeForRecoveryJob(
	ctx context.Context,
	configuration *config.PluginConfiguration,
) (*barmancloudv1.ProbeConfig, error) {
	return impl.collectSidecarProbeForRecoveryJob(ctx, configuration, func(store *barmancloudv1.ObjectStore) *barmancloudv1.ProbeConfig {
		return store.Spec.InstanceSidecarConfiguration.LivenessProbe
	})
}

func (impl LifecycleImplementation) collectSidecarLivenessProbeForInstancePod(
	ctx context.Context,
	configuration *config.PluginConfiguration,
) (*barmancloudv1.ProbeConfig, error) {
	return impl.collectSidecarProbeForInstancePod(ctx, configuration, func(store *barmancloudv1.ObjectStore) *barmancloudv1.ProbeConfig {
		return store.Spec.InstanceSidecarConfiguration.LivenessProbe
	}, "liveness")
}

func (impl LifecycleImplementation) collectSidecarReadinessProbeForRecoveryJob(
	ctx context.Context,
	configuration *config.PluginConfiguration,
) (*barmancloudv1.ProbeConfig, error) {
	return impl.collectSidecarProbeForRecoveryJob(ctx, configuration, func(store *barmancloudv1.ObjectStore) *barmancloudv1.ProbeConfig {
		return store.Spec.InstanceSidecarConfiguration.ReadinessProbe
	})
}

func (impl LifecycleImplementation) collectSidecarReadinessProbeForInstancePod(
	ctx context.Context,
	configuration *config.PluginConfiguration,
) (*barmancloudv1.ProbeConfig, error) {
	return impl.collectSidecarProbeForInstancePod(ctx, configuration, func(store *barmancloudv1.ObjectStore) *barmancloudv1.ProbeConfig {
		return store.Spec.InstanceSidecarConfiguration.ReadinessProbe
	}, "readiness")
}
