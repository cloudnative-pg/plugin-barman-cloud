package main

import (
	"os"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/instance"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	controllerruntime "sigs.k8s.io/controller-runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(barmancloudv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	setupLog.Info("Starting barman cloud instance plugin")
	namespace := os.Getenv("NAMESPACE")
	boName := os.Getenv("BARMAN_OBJECT_NAME")

	mgr, err := controllerruntime.NewManager(controllerruntime.GetConfigOrDie(), controllerruntime.Options{
		Scheme: scheme,
		Cache: cache.Options{
			ByObject: map[client.Object]cache.ByObject{
				&barmancloudv1.ObjectStore{}: {
					Field: fields.OneTermEqualSelector("metadata.name", boName),
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

	if err := mgr.Add(&instance.CNPGI{
		Client: mgr.GetClient(),
		BarmanObjectKey: client.ObjectKey{
			Namespace: namespace,
			Name:      boName,
		},
		// TODO: improve
		PGDataPath:     os.Getenv("PGDATA"),
		PGWALPath:      os.Getenv("PGWAL"),
		SpoolDirectory: os.Getenv("SPOOL_DIRECTORY"),
		ServerCertPath: os.Getenv("SERVER_CERT"),
		ServerKeyPath:  os.Getenv("SERVER_KEY"),
		ClientCertPath: os.Getenv("CLIENT_CERT"),
		ServerAddress:  os.Getenv("SERVER_ADDRESS"),
		PluginPath:     os.Getenv("PLUGIN_PATH"),
	}); err != nil {
		setupLog.Error(err, "unable to create CNPGI runnable")
		os.Exit(1)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
