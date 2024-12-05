package restore

import (
	"context"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
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
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

// Start starts the sidecar informers and CNPG-i server
func Start(ctx context.Context) error {
	setupLog := log.FromContext(ctx)
	setupLog.Info("Starting barman cloud instance plugin")
	namespace := viper.GetString("namespace")
	clusterName := viper.GetString("cluster-name")

	recoveryBarmanObjectName := viper.GetString("recovery-barman-object-name")
	recoveryServerName := viper.GetString("recovery-server-name")

	barmanObjectName := viper.GetString("barman-object-name")
	serverName := viper.GetString("server-name")

	objs := map[client.Object]cache.ByObject{
		&cnpgv1.Cluster{}: {
			Field: fields.OneTermEqualSelector("metadata.name", clusterName),
			Namespaces: map[string]cache.Config{
				namespace: {},
			},
		},
	}

	if recoveryBarmanObjectName != "" {
		objs[&barmancloudv1.ObjectStore{}] = cache.ByObject{
			Field: fields.OneTermEqualSelector("metadata.name", recoveryBarmanObjectName),
			Namespaces: map[string]cache.Config{
				namespace: {},
			},
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Cache: cache.Options{
			ByObject: objs,
		},
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{
					&corev1.Secret{},
					&barmancloudv1.ObjectStore{},
				},
			},
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	if err := mgr.Add(&CNPGI{
		PluginPath:     viper.GetString("plugin-path"),
		SpoolDirectory: viper.GetString("spool-directory"),
		ClusterObjectKey: client.ObjectKey{
			Namespace: namespace,
			Name:      clusterName,
		},
		Client:       mgr.GetClient(),
		PGDataPath:   viper.GetString("pgdata"),
		InstanceName: viper.GetString("pod-name"),

		ServerName: serverName,
		BarmanObjectKey: client.ObjectKey{
			Namespace: namespace,
			Name:      barmanObjectName,
		},

		RecoveryServerName: recoveryServerName,
		RecoveryBarmanObjectKey: client.ObjectKey{
			Namespace: namespace,
			Name:      recoveryBarmanObjectName,
		},
	}); err != nil {
		setupLog.Error(err, "unable to create CNPGI runnable")
		return err
	}

	if err := mgr.Start(ctx); err != nil {
		return err
	}

	return nil
}
