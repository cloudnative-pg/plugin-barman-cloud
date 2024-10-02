package instance

import (
	"context"
	"os"
	"path"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(barmancloudv1.AddToScheme(scheme))
	utilruntime.Must(cnpgv1.AddToScheme(scheme))
}

// Start starts the sidecar informers and CNPG-i server
func Start(ctx context.Context) error {
	setupLog := log.FromContext(ctx)
	setupLog.Info("Starting barman cloud instance plugin")
	namespace := viper.GetString("namespace")
	boName := viper.GetString("barman-object-name")
	clusterName := viper.GetString("cluster-name")
	podName := viper.GetString("pod-name")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Cache: cache.Options{
			ByObject: map[client.Object]cache.ByObject{
				&barmancloudv1.ObjectStore{}: {
					Field: fields.OneTermEqualSelector("metadata.name", boName),
					Namespaces: map[string]cache.Config{
						namespace: {},
					},
				},
				&cnpgv1.Cluster{}: {
					Field: fields.OneTermEqualSelector("metadata.name", clusterName),
					Namespaces: map[string]cache.Config{
						namespace: {},
					},
				},
			},
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := mgr.Add(&CNPGI{
		Client: mgr.GetClient(),
		ClusterObjectKey: client.ObjectKey{
			Namespace: namespace,
			Name:      clusterName,
		},
		BarmanObjectKey: client.ObjectKey{
			Namespace: namespace,
			Name:      boName,
		},
		InstanceName: podName,
		// TODO: improve
		PGDataPath:     viper.GetString("pgdata"),
		PGWALPath:      path.Join(viper.GetString("pgdata"), "pg_wal"),
		SpoolDirectory: viper.GetString("spool-directory"),
		PluginPath:     viper.GetString("plugin-path"),
	}); err != nil {
		setupLog.Error(err, "unable to create CNPGI runnable")
		return err
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}

	return nil
}
