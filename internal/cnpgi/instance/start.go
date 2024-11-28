package instance

import (
	"context"

	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/http"
	"github.com/cloudnative-pg/cnpg-i/pkg/backup"
	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
	"google.golang.org/grpc"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/common"
)

// CNPGI is the implementation of the PostgreSQL sidecar
type CNPGI struct {
	Client           client.Client
	BarmanObjectKey  client.ObjectKey
	ServerName       string
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
		wal.RegisterWALServer(server, common.WALServiceImplementation{
			BarmanObjectKey:  c.BarmanObjectKey,
			ClusterObjectKey: c.ClusterObjectKey,
			ServerName:       c.ServerName,
			InstanceName:     c.InstanceName,
			Client:           c.Client,
			SpoolDirectory:   c.SpoolDirectory,
			PGDataPath:       c.PGDataPath,
			PGWALPath:        c.PGWALPath,
		})
		backup.RegisterBackupServer(server, BackupServiceImplementation{
			Client:           c.Client,
			BarmanObjectKey:  c.BarmanObjectKey,
			ServerName:       c.ServerName,
			ClusterObjectKey: c.ClusterObjectKey,
			InstanceName:     c.InstanceName,
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
