package instance

import (
	"context"
	"path"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
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
	podName := viper.GetString("pod-name")

	controllerOptions := ctrl.Options{
		Scheme: scheme,
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{
					&corev1.Secret{},
					&barmancloudv1.ObjectStore{},
					&cnpgv1.Cluster{},
				},
			},
		},
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), controllerOptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	if err := mgr.Add(&CNPGI{
		Client:       extendedclient.NewExtendedClient(mgr.GetClient()),
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

	if err := mgr.Start(ctx); err != nil {
		return err
	}

	return nil
}
