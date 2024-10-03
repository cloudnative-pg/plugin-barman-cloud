package operator

import (
	"context"

	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/http"
	"github.com/cloudnative-pg/cnpg-i/pkg/lifecycle"
	"github.com/cloudnative-pg/cnpg-i/pkg/reconciler"
	"google.golang.org/grpc"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CNPGI is the implementation of the CNPG-i server
type CNPGI struct {
	Client         client.Client
	PluginPath     string
	ServerCertPath string
	ServerKeyPath  string
	ClientCertPath string
	ServerAddress  string
}

// Start starts the GRPC server
// of the operator plugin
func (c *CNPGI) Start(ctx context.Context) error {
	enrich := func(server *grpc.Server) error {
		reconciler.RegisterReconcilerHooksServer(server, ReconcilerImplementation{
			Client: c.Client,
		})
		lifecycle.RegisterOperatorLifecycleServer(server, LifecycleImplementation{})
		return nil
	}

	srv := http.Server{
		IdentityImpl:   IdentityImplementation{},
		Enrichers:      []http.ServerEnricher{enrich},
		PluginPath:     c.PluginPath,
		ServerCertPath: c.ServerCertPath,
		ServerKeyPath:  c.ServerKeyPath,
		ClientCertPath: c.ClientCertPath,
		ServerAddress:  c.ServerAddress,
	}

	return srv.Start(ctx)
}
