package restore

import (
	"context"
	"path"

	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/http"
	restore "github.com/cloudnative-pg/cnpg-i/pkg/restore/job"
	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
	"google.golang.org/grpc"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/common"
)

// CNPGI is the implementation of the PostgreSQL sidecar
type CNPGI struct {
	PluginPath     string
	SpoolDirectory string

	Client       client.Client
	PGDataPath   string
	InstanceName string
}

// Start starts the GRPC service
func (c *CNPGI) Start(ctx context.Context) error {
	// PgWalVolumePgWalPath is the path of pg_wal directory inside the WAL volume when present
	const PgWalVolumePgWalPath = "/var/lib/postgresql/wal/pg_wal"

	enrich := func(server *grpc.Server) error {
		wal.RegisterWALServer(server, common.WALServiceImplementation{
			InstanceName:   c.InstanceName,
			Client:         c.Client,
			SpoolDirectory: c.SpoolDirectory,
			PGDataPath:     c.PGDataPath,
			PGWALPath:      path.Join(c.PGDataPath, "pg_wal"),
		})

		restore.RegisterRestoreJobHooksServer(server, &JobHookImpl{
			Client:               c.Client,
			SpoolDirectory:       c.SpoolDirectory,
			PgDataPath:           c.PGDataPath,
			PgWalFolderToSymlink: PgWalVolumePgWalPath,
		})

		common.AddHealthCheck(server)

		return nil
	}

	srv := http.Server{
		IdentityImpl: IdentityImplementation{},
		Enrichers:    []http.ServerEnricher{enrich},
		PluginPath:   c.PluginPath,
	}

	return srv.Start(ctx)
}
