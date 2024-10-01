package operator

import (
	"context"

	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/http"
	"github.com/cloudnative-pg/cnpg-i/pkg/identity"
	"github.com/cloudnative-pg/cnpg-i/pkg/reconciler"
	"google.golang.org/grpc"
)

// CNPGI is the implementation of the Operator plugin
type CNPGI struct{}

// Start starts the GRPC server
func (c *CNPGI) Start(ctx context.Context) error {
	cmd := http.CreateMainCmd(IdentityImplementation{}, func(server *grpc.Server) error {
		// Register the declared implementations
		identity.RegisterIdentityServer(server, IdentityImplementation{})
		reconciler.RegisterReconcilerHooksServer(server, ReconcilerImplementation{})
		return nil
	})
	cmd.Use = "plugin-operator"

	return cmd.ExecuteContext(ctx) //nolint:wrapcheck
}
