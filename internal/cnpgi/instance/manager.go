package instance

import (
	"context"
	"path"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	extendedclient "github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/instance/internal/client"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(barmancloudv1.AddToScheme(scheme))
	utilruntime.Must(cnpgv1.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

// Start starts the sidecar informers and CNPG-i server
func Start(ctx context.Context) error {
	setupLog := log.FromContext(ctx)
	setupLog.Info("Starting barman cloud instance plugin")
	namespace := viper.GetString("namespace")
	clusterName := viper.GetString("cluster-name")
	podName := viper.GetString("pod-name")

	barmanObjectName := viper.GetString("barman-object-name")
	recoveryBarmanObjectName := viper.GetString("recovery-barman-object-name")

	controllerOptions := ctrl.Options{
		Scheme: scheme,
		Cache: cache.Options{
			ByObject: map[client.Object]cache.ByObject{
				&cnpgv1.Cluster{}: {
					Field: fields.OneTermEqualSelector("metadata.name", clusterName),
					Namespaces: map[string]cache.Config{
						namespace: {},
					},
				},
			},
		},
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{
					&corev1.Secret{},
				},
			},
		},
	}

	if len(recoveryBarmanObjectName) == 0 {
		controllerOptions.Cache.ByObject[&barmancloudv1.ObjectStore{}] = cache.ByObject{
			Field: fields.OneTermEqualSelector("metadata.name", barmanObjectName),
			Namespaces: map[string]cache.Config{
				namespace: {},
			},
		}
	} else {
		controllerOptions.Client.Cache.DisableFor = append(
			controllerOptions.Client.Cache.DisableFor,
			&barmancloudv1.ObjectStore{},
		)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), controllerOptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	barmanObjectKey := client.ObjectKey{
		Namespace: namespace,
		Name:      barmanObjectName,
	}
	recoveryBarmanObjectKey := client.ObjectKey{
		Namespace: namespace,
		Name:      recoveryBarmanObjectName,
	}

	involvedObjectStores := make([]types.NamespacedName, 0, 2)
	if len(barmanObjectName) > 0 {
		involvedObjectStores = append(involvedObjectStores, barmanObjectKey)
	}
	if len(recoveryBarmanObjectName) > 0 {
		involvedObjectStores = append(involvedObjectStores, recoveryBarmanObjectKey)
	}

	if err := mgr.Add(&CNPGI{
		Client: extendedclient.NewExtendedClient(mgr.GetClient(), involvedObjectStores),
		ClusterObjectKey: client.ObjectKey{
			Namespace: namespace,
			Name:      clusterName,
		},
		InstanceName: podName,
		// TODO: improve
		PGDataPath:     viper.GetString("pgdata"),
		PGWALPath:      path.Join(viper.GetString("pgdata"), "pg_wal"),
		SpoolDirectory: viper.GetString("spool-directory"),
		PluginPath:     viper.GetString("plugin-path"),

		BarmanObjectKey: barmanObjectKey,
		ServerName:      viper.GetString("server-name"),

		RecoveryBarmanObjectKey: recoveryBarmanObjectKey,
		RecoveryServerName:      viper.GetString("recovery-server-name"),
	}); err != nil {
		setupLog.Error(err, "unable to create CNPGI runnable")
		return err
	}

	if err := mgr.Start(ctx); err != nil {
		return err
	}

	return nil
}
