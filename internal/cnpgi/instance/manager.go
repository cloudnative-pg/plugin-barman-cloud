package instance

import (
	"context"
	"path"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/machinery/pkg/log"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	extendedclient "github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/instance/internal/client"
)

// Start starts the sidecar informers and CNPG-i server
func Start(ctx context.Context) error {
	scheme := generateScheme(ctx)

	setupLog := log.FromContext(ctx)
	setupLog.Info("Starting barman cloud instance plugin")

	podName := viper.GetString("pod-name")
	clusterName := viper.GetString("cluster-name")
	namespace := viper.GetString("namespace")

	controllerOptions := ctrl.Options{
		Scheme: scheme,
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{
					&corev1.Secret{},
					&barmancloudv1.ObjectStore{},
					&cnpgv1.Cluster{},
					&cnpgv1.Backup{},
				},
			},
		},
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), controllerOptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	customCacheClient := extendedclient.NewExtendedClient(mgr.GetClient())

	if err := mgr.Add(&CNPGI{
		Client:         customCacheClient,
		InstanceName:   podName,
		PGDataPath:     viper.GetString("pgdata"),
		PGWALPath:      path.Join(viper.GetString("pgdata"), "pg_wal"),
		SpoolDirectory: viper.GetString("spool-directory"),
		PluginPath:     viper.GetString("plugin-path"),
	}); err != nil {
		setupLog.Error(err, "unable to create CNPGI runnable")
		return err
	}

	if err := mgr.Add(&CatalogMaintenanceRunnable{
		Client:   customCacheClient,
		Recorder: mgr.GetEventRecorderFor("policy-runnable"),
		ClusterKey: types.NamespacedName{
			Namespace: namespace,
			Name:      clusterName,
		},
		CurrentPodName: podName,
	}); err != nil {
		setupLog.Error(err, "unable to policy enforcement runnable")
		return err
	}

	if err := mgr.Start(ctx); err != nil {
		return err
	}

	return nil
}

// generateScheme creates a runtime.Scheme object with all the
// definition needed to support the sidecar. This allows
// the plugin to be used in every CNPG-based operator.
func generateScheme(ctx context.Context) *runtime.Scheme {
	result := runtime.NewScheme()

	utilruntime.Must(barmancloudv1.AddToScheme(result))
	utilruntime.Must(clientgoscheme.AddToScheme(result))

	cnpgGroup := viper.GetString("custom-cnpg-group")
	cnpgVersion := viper.GetString("custom-cnpg-version")
	if len(cnpgGroup) == 0 {
		cnpgGroup = cnpgv1.SchemeGroupVersion.Group
	}
	if len(cnpgVersion) == 0 {
		cnpgVersion = cnpgv1.SchemeGroupVersion.Version
	}

	// Proceed with custom registration of the CNPG scheme
	schemeGroupVersion := schema.GroupVersion{Group: cnpgGroup, Version: cnpgVersion}
	schemeBuilder := &scheme.Builder{GroupVersion: schemeGroupVersion}
	schemeBuilder.Register(&cnpgv1.Cluster{}, &cnpgv1.ClusterList{})
	schemeBuilder.Register(&cnpgv1.Backup{}, &cnpgv1.BackupList{})
	schemeBuilder.Register(&cnpgv1.ScheduledBackup{}, &cnpgv1.ScheduledBackupList{})
	utilruntime.Must(schemeBuilder.AddToScheme(result))

	schemeLog := log.FromContext(ctx)
	schemeLog.Info("CNPG types registration", "schemeGroupVersion", schemeGroupVersion)

	return result
}
