package operator

import (
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/http"
	"github.com/cloudnative-pg/cnpg-i/pkg/lifecycle"
	"github.com/cloudnative-pg/cnpg-i/pkg/reconciler"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

// NewCommand creates the command to start the GRPC server
// of the operator plugin
func NewCommand() *cobra.Command {
	cmd := http.CreateMainCmd(IdentityImplementation{}, func(server *grpc.Server) error {
		reconciler.RegisterReconcilerHooksServer(server, ReconcilerImplementation{})
		lifecycle.RegisterOperatorLifecycleServer(server, LifecycleImplementation{})
		return nil
	})
	cmd.Use = "plugin"
	return cmd
}
