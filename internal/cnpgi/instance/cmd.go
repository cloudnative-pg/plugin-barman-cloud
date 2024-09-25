package instance

import (
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/http"
	"github.com/cloudnative-pg/cnpg-i/pkg/backup"
	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

func NewCMD() *cobra.Command {
	cmd := http.CreateMainCmd(IdentityImplementation{}, func(server *grpc.Server) error {
		// Register the declared implementations
		wal.RegisterWALServer(server, WALServiceImplementation{})
		backup.RegisterBackupServer(server, BackupServiceImplementation{})
		return nil
	})

	cmd.Use = "plugin-instance"

	return cmd
}
