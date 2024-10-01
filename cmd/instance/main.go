// Package main is the implementation of the CNPG-i PostgreSQL sidecar
package main

import (
	"errors"
	"os"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/instance"
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
	namespace := mustGetEnv("NAMESPACE")
	boName := mustGetEnv("BARMAN_OBJECT_NAME")
	clusterName := mustGetEnv("CLUSTER_NAME")
	instanceName := mustGetEnv("INSTANCE_NAME")

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

	if err := mgr.Add(&instance.CNPGI{
		Client: mgr.GetClient(),
		ClusterObjectKey: client.ObjectKey{
			Namespace: namespace,
			Name:      clusterName,
		},
		BarmanObjectKey: client.ObjectKey{
			Namespace: namespace,
			Name:      boName,
		},
		InstanceName: instanceName,
		// TODO: improve
		PGDataPath:     mustGetEnv("PGDATA"),
		PGWALPath:      mustGetEnv("PGWAL"),
		SpoolDirectory: mustGetEnv("SPOOL_DIRECTORY"),
		ServerCertPath: mustGetEnv("SERVER_CERT"),
		ServerKeyPath:  mustGetEnv("SERVER_KEY"),
		ClientCertPath: mustGetEnv("CLIENT_CERT"),
		ServerAddress:  mustGetEnv("SERVER_ADDRESS"),
		PluginPath:     mustGetEnv("PLUGIN_PATH"),
	}); err != nil {
		setupLog.Error(err, "unable to create CNPGI runnable")
		os.Exit(1)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func mustGetEnv(envName string) string {
	value := os.Getenv(envName)
	if value == "" {
		setupLog.Error(
			errors.New("missing required env variable"),
			"while fetching env variables",
			"name",
			envName,
		)
		os.Exit(1)
	}
	return value
}
