package instance

import (
	"context"

	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/http"
	"github.com/cloudnative-pg/cnpg-i/pkg/backup"
	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
	"google.golang.org/grpc"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CNPGI struct {
	Client          client.Client
	BarmanObjectKey client.ObjectKey
	PGDataPath      string
	PGWALPath       string
	SpoolDirectory  string
	ServerCertPath  string
	ServerKeyPath   string
	ClientCertPath  string
	// mutually exclusive with pluginPath
	ServerAddress string
	// mutually exclusive with serverAddress
	PluginPath string
}

func (c *CNPGI) Start(ctx context.Context) error {
	enrich := func(server *grpc.Server) error {
		wal.RegisterWALServer(server, WALServiceImplementation{
			BarmanObjectKey: c.BarmanObjectKey,
			Client:          c.Client,
			SpoolDirectory:  c.SpoolDirectory,
			PGDataPath:      c.PGDataPath,
			PGWALPath:       c.PGWALPath,
		})
		backup.RegisterBackupServer(server, BackupServiceImplementation{})
		return nil
	}

	srv := http.Server{
		IdentityImpl: IdentityImplementation{
			Client:          c.Client,
			BarmanObjectKey: c.BarmanObjectKey,
		},
		Enrichers:      []http.ServerEnricher{enrich},
		ServerCertPath: c.ServerCertPath,
		ServerKeyPath:  c.ServerKeyPath,
		ClientCertPath: c.ClientCertPath,
		ServerAddress:  c.ServerAddress,
		PluginPath:     c.PluginPath,
	}

	return srv.Start(ctx)
}
