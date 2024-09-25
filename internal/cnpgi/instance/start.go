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
		IdentityImpl: IdentityImplementation{},
		Enrichers:    []http.ServerEnricher{enrich},
		// TODO: fille
		ServerCertPath: "",
		ServerKeyPath:  "",
		ClientCertPath: "",
		ServerAddress:  "",
		PluginPath:     "",
	}

	return srv.Start(ctx)
}
