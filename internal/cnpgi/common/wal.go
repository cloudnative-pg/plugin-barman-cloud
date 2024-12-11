package common

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/cloudnative-pg/barman-cloud/pkg/archiver"
	barmanCommand "github.com/cloudnative-pg/barman-cloud/pkg/command"
	barmanCredentials "github.com/cloudnative-pg/barman-cloud/pkg/credentials"
	barmanRestorer "github.com/cloudnative-pg/barman-cloud/pkg/restorer"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
	"github.com/cloudnative-pg/machinery/pkg/log"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"
)

// WALServiceImplementation is the implementation of the WAL Service
type WALServiceImplementation struct {
	wal.UnimplementedWALServer
	Client         client.Client
	InstanceName   string
	SpoolDirectory string
	PGDataPath     string
	PGWALPath      string
}

// GetCapabilities implements the WALService interface
func (w WALServiceImplementation) GetCapabilities(
	_ context.Context,
	_ *wal.WALCapabilitiesRequest,
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

// Archive implements the WALService interface
func (w WALServiceImplementation) Archive(
	ctx context.Context,
	request *wal.WALArchiveRequest,
) (*wal.WALArchiveResult, error) {
	contextLogger := log.FromContext(ctx)
	contextLogger.Debug("starting wal archive")

	configuration, err := config.NewFromClusterJSON(request.ClusterDefinition)
	if err != nil {
		return nil, err
	}

	var objectStore barmancloudv1.ObjectStore
	if err := w.Client.Get(ctx, configuration.GetBarmanObjectKey(), &objectStore); err != nil {
		return nil, err
	}

	envArchive, err := barmanCredentials.EnvSetBackupCloudCredentials(
		ctx,
		w.Client,
		objectStore.Namespace,
		&objectStore.Spec.Configuration,
		os.Environ())
	if err != nil {
		if apierrors.IsForbidden(err) {
			return nil, errors.New("backup credentials don't yet have access permissions. Will retry reconciliation loop")
		}
		return nil, err
	}

	arch, err := archiver.New(
		ctx,
		envArchive,
		w.SpoolDirectory,
		w.PGDataPath,
		path.Join(w.PGDataPath, metadata.CheckEmptyWalArchiveFile),
	)
	if err != nil {
		return nil, err
	}

	options, err := arch.BarmanCloudWalArchiveOptions(ctx, &objectStore.Spec.Configuration, configuration.ServerName)
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

// Restore implements the WALService interface
// nolint: gocognit
func (w WALServiceImplementation) Restore(
	ctx context.Context,
	request *wal.WALRestoreRequest,
) (*wal.WALRestoreResult, error) {
	contextLogger := log.FromContext(ctx)

	walName := request.GetSourceWalName()
	destinationPath := request.GetDestinationFileName()

	configuration, err := config.NewFromClusterJSON(request.ClusterDefinition)
	if err != nil {
		return nil, err
	}

	var serverName string
	var objectStoreKey types.NamespacedName

	var promotionToken string
	if configuration.Cluster.Spec.ReplicaCluster != nil {
		promotionToken = configuration.Cluster.Spec.ReplicaCluster.PromotionToken
	}

	switch {
	case promotionToken != "" && configuration.Cluster.Status.LastPromotionToken != promotionToken:
		// This is a replica cluster that is being promoted to a primary cluster
		// Recover from the replica source object store
		serverName = configuration.ReplicaSourceServerName
		objectStoreKey = configuration.GetReplicaSourceBarmanObjectKey()

	case configuration.Cluster.IsReplica() && configuration.Cluster.Status.CurrentPrimary == w.InstanceName:
		// Designated primary on replica cluster, using replica source object store
		serverName = configuration.ReplicaSourceServerName
		objectStoreKey = configuration.GetReplicaSourceBarmanObjectKey()

	case configuration.Cluster.Status.CurrentPrimary == "":
		// Recovery from object store, using recovery object store
		serverName = configuration.RecoveryServerName
		objectStoreKey = configuration.GetRecoveryBarmanObjectKey()

	default:
		// Using cluster object store
		serverName = configuration.ServerName
		objectStoreKey = configuration.GetBarmanObjectKey()
	}

	var objectStore barmancloudv1.ObjectStore
	if err := w.Client.Get(ctx, objectStoreKey, &objectStore); err != nil {
		return nil, err
	}

	contextLogger.Info(
		"Restoring WAL file",
		"objectStore", objectStore.Name,
		"serverName", serverName,
		"walName", walName)
	return &wal.WALRestoreResult{}, w.restoreFromBarmanObjectStore(
		ctx, configuration.Cluster, &objectStore, serverName, walName, destinationPath)
}

func (w WALServiceImplementation) restoreFromBarmanObjectStore(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	objectStore *barmancloudv1.ObjectStore,
	serverName string,
	walName string,
	destinationPath string,
) error {
	contextLogger := log.FromContext(ctx)
	startTime := time.Now()

	barmanConfiguration := &objectStore.Spec.Configuration

	env := GetRestoreCABundleEnv(barmanConfiguration)
	credentialsEnv, err := barmanCredentials.EnvSetBackupCloudCredentials(
		ctx,
		w.Client,
		objectStore.Namespace,
		&objectStore.Spec.Configuration,
		os.Environ(),
	)
	if err != nil {
		return fmt.Errorf("while getting recover credentials: %w", err)
	}
	env = MergeEnv(env, credentialsEnv)

	options, err := barmanCommand.CloudWalRestoreOptions(ctx, barmanConfiguration, serverName)
	if err != nil {
		return fmt.Errorf("while getting barman-cloud-wal-restore options: %w", err)
	}

	// Create the restorer
	var walRestorer *barmanRestorer.WALRestorer
	if walRestorer, err = barmanRestorer.New(ctx, env, w.SpoolDirectory); err != nil {
		return fmt.Errorf("while creating the restorer: %w", err)
	}

	// Step 1: check if this WAL file is not already in the spool
	var wasInSpool bool
	if wasInSpool, err = walRestorer.RestoreFromSpool(walName, destinationPath); err != nil {
		return fmt.Errorf("while restoring a file from the spool directory: %w", err)
	}
	if wasInSpool {
		contextLogger.Info("Restored WAL file from spool (parallel)",
			"walName", walName,
		)
		return nil
	}

	// We skip this step if streaming connection is not available
	if isStreamingAvailable(cluster, w.InstanceName) {
		if err := checkEndOfWALStreamFlag(walRestorer); err != nil {
			return err
		}
	}

	// Step 3: gather the WAL files names to restore. If the required file isn't a regular WAL, we download it directly.
	var walFilesList []string
	maxParallel := 1
	if barmanConfiguration.Wal != nil && barmanConfiguration.Wal.MaxParallel > 1 {
		maxParallel = barmanConfiguration.Wal.MaxParallel
	}
	if IsWALFile(walName) {
		// If this is a regular WAL file, we try to prefetch
		if walFilesList, err = gatherWALFilesToRestore(walName, maxParallel); err != nil {
			return fmt.Errorf("while generating the list of WAL files to restore: %w", err)
		}
	} else {
		// This is not a regular WAL file, we fetch it directly
		walFilesList = []string{walName}
	}

	// Step 4: download the WAL files into the required place
	downloadStartTime := time.Now()
	walStatus := walRestorer.RestoreList(ctx, walFilesList, destinationPath, options)

	// We return immediately if the first WAL has errors, because the first WAL
	// is the one that PostgreSQL has requested to restore.
	// The failure has already been logged in walRestorer.RestoreList method
	if walStatus[0].Err != nil {
		if errors.Is(walStatus[0].Err, barmanRestorer.ErrWALNotFound) {
			return newWALNotFoundError()
		}

		return walStatus[0].Err
	}

	// We skip this step if streaming connection is not available
	endOfWALStream := isEndOfWALStream(walStatus)
	if isStreamingAvailable(cluster, w.InstanceName) && endOfWALStream {
		contextLogger.Info(
			"Set end-of-wal-stream flag as one of the WAL files to be prefetched was not found")

		err = walRestorer.SetEndOfWALStream()
		if err != nil {
			return err
		}
	}

	successfulWalRestore := 0
	for idx := range walStatus {
		if walStatus[idx].Err == nil {
			successfulWalRestore++
		}
	}

	contextLogger.Info("WAL restore command completed (parallel)",
		"walName", walName,
		"maxParallel", maxParallel,
		"successfulWalRestore", successfulWalRestore,
		"failedWalRestore", maxParallel-successfulWalRestore,
		"startTime", startTime,
		"downloadStartTime", downloadStartTime,
		"downloadTotalTime", time.Since(downloadStartTime),
		"totalTime", time.Since(startTime))

	return nil
}

// Status implements the WALService interface
func (w WALServiceImplementation) Status(
	_ context.Context,
	_ *wal.WALStatusRequest,
) (*wal.WALStatusResult, error) {
	// TODO implement me
	panic("implement me")
}

// SetFirstRequired implements the WALService interface
func (w WALServiceImplementation) SetFirstRequired(
	_ context.Context,
	_ *wal.SetFirstRequiredRequest,
) (*wal.SetFirstRequiredResult, error) {
	// TODO implement me
	panic("implement me")
}

// isStreamingAvailable checks if this pod can replicate via streaming connection.
func isStreamingAvailable(cluster *cnpgv1.Cluster, podName string) bool {
	if cluster == nil {
		return false
	}

	// Easy case: If this pod is a replica, the streaming is always available
	if cluster.Status.CurrentPrimary != podName {
		return true
	}

	// Designated primary in a replica cluster: return true if the external cluster has streaming connection
	if cluster.IsReplica() {
		externalCluster, found := cluster.ExternalCluster(cluster.Spec.ReplicaCluster.Source)

		// This is a configuration error
		if !found {
			return false
		}

		return externalCluster.ConnectionParameters != nil
	}

	// Primary, we do not replicate from nobody
	return false
}

// gatherWALFilesToRestore files a list of possible WAL files to restore, always
// including as the first one the requested WAL file.
func gatherWALFilesToRestore(walName string, parallel int) (walList []string, err error) {
	var segment Segment

	segment, err = SegmentFromName(walName)
	if err != nil {
		// This seems an invalid segment name. It's not a problem
		// because PostgreSQL may request also other files such as
		// backup, history, etc.
		// Let's just avoid prefetching in this case
		return []string{walName}, nil
	}
	// NextSegments would accept postgresVersion and segmentSize,
	// but we do not have this info here, so we pass nil.
	segmentList := segment.NextSegments(parallel, nil, nil)
	walList = make([]string, len(segmentList))
	for idx := range segmentList {
		walList[idx] = segmentList[idx].Name()
	}

	return walList, err
}

// ErrEndOfWALStreamReached is returned when end of WAL is detected in the cloud archive.
var ErrEndOfWALStreamReached = errors.New("end of WAL reached")

// checkEndOfWALStreamFlag returns ErrEndOfWALStreamReached if the flag is set in the restorer.
func checkEndOfWALStreamFlag(walRestorer *barmanRestorer.WALRestorer) error {
	contain, err := walRestorer.IsEndOfWALStream()
	if err != nil {
		return err
	}

	if contain {
		err := walRestorer.ResetEndOfWalStream()
		if err != nil {
			return err
		}

		return ErrEndOfWALStreamReached
	}
	return nil
}

// isEndOfWALStream returns true if one of the downloads has returned
// a file-not-found error.
func isEndOfWALStream(results []barmanRestorer.Result) bool {
	for _, result := range results {
		if errors.Is(result.Err, barmanRestorer.ErrWALNotFound) {
			return true
		}
	}

	return false
}
