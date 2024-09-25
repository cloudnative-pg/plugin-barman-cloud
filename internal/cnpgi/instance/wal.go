package instance

import (
	"context"

	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
)

type WALServiceImplementation struct {
	wal.UnimplementedWALServer
}

func (W WALServiceImplementation) GetCapabilities(ctx context.Context, request *wal.WALCapabilitiesRequest) (*wal.WALCapabilitiesResult, error) {
	return &wal.WALCapabilitiesResult{
		Capabilities: []*wal.WALCapability{
			{
				Type: &wal.WALCapability_Rpc{
					Rpc: &wal.WALCapability_RPC{
						Type: wal.WALCapability_RPC_TYPE_ARCHIVE_WAL,
					},
				},
			},
			{
				Type: &wal.WALCapability_Rpc{
					Rpc: &wal.WALCapability_RPC{
						Type: wal.WALCapability_RPC_TYPE_RESTORE_WAL,
					},
				},
			},
		},
	}, nil
}

func (W WALServiceImplementation) Archive(ctx context.Context, request *wal.WALArchiveRequest) (*wal.WALArchiveResult, error) {
	// TODO implement me
	panic("implement me")
}

func (W WALServiceImplementation) Restore(ctx context.Context, request *wal.WALRestoreRequest) (*wal.WALRestoreResult, error) {
	// TODO implement me
	panic("implement me")
}

func (W WALServiceImplementation) Status(ctx context.Context, request *wal.WALStatusRequest) (*wal.WALStatusResult, error) {
	// TODO implement me
	panic("implement me")
}

func (W WALServiceImplementation) SetFirstRequired(ctx context.Context, request *wal.SetFirstRequiredRequest) (*wal.SetFirstRequiredResult, error) {
	// TODO implement me
	panic("implement me")
}
