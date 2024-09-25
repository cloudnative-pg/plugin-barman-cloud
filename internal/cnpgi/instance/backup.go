package instance

import (
	"context"

	"github.com/cloudnative-pg/cnpg-i/pkg/backup"
)

type BackupServiceImplementation struct {
	backup.UnimplementedBackupServer
}

func (b BackupServiceImplementation) GetCapabilities(ctx context.Context, request *backup.BackupCapabilitiesRequest) (*backup.BackupCapabilitiesResult, error) {
	return &backup.BackupCapabilitiesResult{
		Capabilities: []*backup.BackupCapability{
			{
				Type: &backup.BackupCapability_Rpc{
					Rpc: &backup.BackupCapability_RPC{
						Type: backup.BackupCapability_RPC_TYPE_BACKUP,
					},
				},
			},
		},
	}, nil
}

func (b BackupServiceImplementation) Backup(ctx context.Context, request *backup.BackupRequest) (*backup.BackupResult, error) {
	// TODO implement me
	panic("implement me")
}
