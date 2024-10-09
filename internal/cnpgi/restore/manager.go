package restore

import (
	"context"
	"os"

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
	archiveConfiguration := viper.GetString("barman-archive-configuration")
	clusterName := viper.GetString("cluster-name")
	backupToRestoreName := viper.GetString("backup-to-restore")

	objs := map[client.Object]cache.ByObject{
		&cnpgv1.Cluster{}: {
			Field: fields.OneTermEqualSelector("metadata.name", clusterName),
			Namespaces: map[string]cache.Config{
				namespace: {},
			},
		},
		&cnpgv1.Backup{}: {
			Field: fields.OneTermEqualSelector("metadata.name", backupToRestoreName),
			Namespaces: map[string]cache.Config{
				namespace: {},
			},
		},
	}
	if archiveConfiguration != "" {
		objs[&barmancloudv1.ObjectStore{}] = cache.ByObject{
			Field: fields.OneTermEqualSelector("metadata.name", archiveConfiguration),
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
				},
			},
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := mgr.Add(&CNPGI{
		PluginPath:     viper.GetString("plugin-path"),
		SpoolDirectory: viper.GetString("spool-directory"),
		ArchiveConfiguration: client.ObjectKey{
			Namespace: namespace,
			Name:      archiveConfiguration,
		},
		ClusterObjectKey: client.ObjectKey{
			Namespace: namespace,
			Name:      clusterName,
		},
		BackupToRestoreObjectKey: client.ObjectKey{
			Namespace: namespace,
			Name:      backupToRestoreName,
		},
		Client:     mgr.GetClient(),
		PGDataPath: viper.GetString("pgdata"),
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
