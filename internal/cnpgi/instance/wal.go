package instance

import (
	"context"
	"errors"
	barmanCredentials "github.com/cloudnative-pg/barman-cloud/pkg/credentials"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"os"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudnative-pg/barman-cloud/pkg/archiver"
	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
)

type WALServiceImplementation struct {
	BarmanObjectKey client.ObjectKey
	Client          client.Client
	SpoolDirectory  string
	PGDataPath      string
	PGWALPath       string
	wal.UnimplementedWALServer
}

func (w WALServiceImplementation) GetCapabilities(ctx context.Context, request *wal.WALCapabilitiesRequest) (*wal.WALCapabilitiesResult, error) {
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

func (w WALServiceImplementation) Archive(ctx context.Context, request *wal.WALArchiveRequest) (*wal.WALArchiveResult, error) {
	var objectStore barmancloudv1.ObjectStore
	if err := w.Client.Get(ctx, w.BarmanObjectKey, &objectStore); err != nil {
		return nil, err
	}

	envArchive, err := barmanCredentials.EnvSetBackupCloudCredentials(
		ctx,
		w.Client,
		objectStore.Namespace,
		&objectStore.Spec.Configuration,
		os.Environ())
	if apierrors.IsForbidden(err) {
		return nil, errors.New("backup credentials don't yet have access permissions. Will retry reconciliation loop")
	}

	arch, err := archiver.New(ctx, envArchive, w.SpoolDirectory, w.PGDataPath, w.PGWALPath)
	if err != nil {
		return nil, err
	}

	options, err := arch.BarmanCloudWalArchiveOptions(ctx, &objectStore.Spec.Configuration, objectStore.Name)
	if err != nil {
		return nil, err
	}
	walList := arch.GatherWALFilesToArchive(ctx, request.GetSourceFileName(), 1)
	result := arch.ArchiveList(ctx, walList, options)
	for _, archiverResult := range result {
		if archiverResult.Err != nil {
			return nil, archiverResult.Err
		}
	}

	return &wal.WALArchiveResult{}, nil
}

func (w WALServiceImplementation) Restore(ctx context.Context, request *wal.WALRestoreRequest) (*wal.WALRestoreResult, error) {
	// TODO implement me
	panic("implement me")
}

func (w WALServiceImplementation) Status(ctx context.Context, request *wal.WALStatusRequest) (*wal.WALStatusResult, error) {
	// TODO implement me
	panic("implement me")
}

func (w WALServiceImplementation) SetFirstRequired(ctx context.Context, request *wal.SetFirstRequiredRequest) (*wal.SetFirstRequiredResult, error) {
	// TODO implement me
	panic("implement me")
}
