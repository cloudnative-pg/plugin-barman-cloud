package restore

import (
	"context"

	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/http"
	restore "github.com/cloudnative-pg/cnpg-i/pkg/restore/job"
	"google.golang.org/grpc"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CNPGI is the implementation of the PostgreSQL sidecar
type CNPGI struct {
	PluginPath       string
	SpoolDirectory   string
	ClusterObjectKey client.ObjectKey
	Client           client.Client
	PGDataPath       string
}

// Start starts the GRPC service
func (c *CNPGI) Start(ctx context.Context) error {
	// PgWalVolumePgWalPath is the path of pg_wal directory inside the WAL volume when present
	const PgWalVolumePgWalPath = "/var/lib/postgresql/wal/pg_wal"

	enrich := func(server *grpc.Server) error {
		restore.RegisterRestoreJobHooksServer(server, &JobHookImpl{
			Client:               c.Client,
			ClusterObjectKey:     c.ClusterObjectKey,
			SpoolDirectory:       c.SpoolDirectory,
			PgDataPath:           c.PGDataPath,
			PgWalFolderToSymlink: PgWalVolumePgWalPath,
		})
		return nil
	}

	srv := http.Server{
		IdentityImpl: IdentityImplementation{},
		Enrichers:    []http.ServerEnricher{enrich},
		PluginPath:   c.PluginPath,
	}

	return srv.Start(ctx)
}
