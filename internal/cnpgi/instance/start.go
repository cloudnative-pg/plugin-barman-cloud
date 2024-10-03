package instance

import (
	"context"

	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/http"
	"github.com/cloudnative-pg/cnpg-i/pkg/backup"
	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
	"google.golang.org/grpc"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CNPGI is the implementation of the PostgreSQL sidecar
type CNPGI struct {
	Client           client.Client
	BarmanObjectKey  client.ObjectKey
	ClusterObjectKey client.ObjectKey
	PGDataPath       string
	PGWALPath        string
	SpoolDirectory   string
	// mutually exclusive with serverAddress
	PluginPath   string
	InstanceName string
}

// Start starts the GRPC service
func (c *CNPGI) Start(ctx context.Context) error {
	enrich := func(server *grpc.Server) error {
		wal.RegisterWALServer(server, WALServiceImplementation{
			BarmanObjectKey:  c.BarmanObjectKey,
			ClusterObjectKey: c.ClusterObjectKey,
			InstanceName:     c.InstanceName,
			Client:           c.Client,
			SpoolDirectory:   c.SpoolDirectory,
			PGDataPath:       c.PGDataPath,
			PGWALPath:        c.PGWALPath,
		})
		backup.RegisterBackupServer(server, BackupServiceImplementation{
			Client:       c.Client,
			InstanceName: c.InstanceName,
		})
		return nil
	}

	srv := http.Server{
		IdentityImpl: IdentityImplementation{
			Client:          c.Client,
			BarmanObjectKey: c.BarmanObjectKey,
		},
		Enrichers:  []http.ServerEnricher{enrich},
		PluginPath: c.PluginPath,
	}

	return srv.Start(ctx)
}
