package instance

import (
	"context"

	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
)

type WALServiceImplementation struct {
	wal.UnimplementedWALServer
}

func (w WALServiceImplementation) GetCapabilities(
	_ context.Context, _ *wal.WALCapabilitiesRequest,
) (*wal.WALCapabilitiesResult, error) {
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

func (w WALServiceImplementation) Archive(_ context.Context, _ *wal.WALArchiveRequest) (*wal.WALArchiveResult,
	error,
) {
	// TODO implement me
	panic("implement me")
}

func (w WALServiceImplementation) Restore(_ context.Context, _ *wal.WALRestoreRequest) (*wal.WALRestoreResult,
	error,
) {
	// TODO implement me
	panic("implement me")
}

func (w WALServiceImplementation) Status(_ context.Context, _ *wal.WALStatusRequest) (*wal.WALStatusResult,
	error,
) {
	// TODO implement me
	panic("implement me")
}

func (w WALServiceImplementation) SetFirstRequired(
	_ context.Context, _ *wal.SetFirstRequiredRequest,
) (*wal.SetFirstRequiredResult, error) {
	// TODO implement me
	panic("implement me")
}
