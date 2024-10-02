package instance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	barmanapi "github.com/cloudnative-pg/barman-cloud/pkg/api"
	"github.com/cloudnative-pg/barman-cloud/pkg/archiver"
	barmanCommand "github.com/cloudnative-pg/barman-cloud/pkg/command"
	barmanCredentials "github.com/cloudnative-pg/barman-cloud/pkg/credentials"
	barmanRestorer "github.com/cloudnative-pg/barman-cloud/pkg/restorer"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
	"github.com/cloudnative-pg/machinery/pkg/log"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
)

const (
	// CheckEmptyWalArchiveFile is the name of the file in the PGDATA that,
	// if present, requires the WAL archiver to check that the backup object
	// store is empty.
	CheckEmptyWalArchiveFile = ".check-empty-wal-archive"
)

// WALServiceImplementation is the implementation of the WAL Service
type WALServiceImplementation struct {
	BarmanObjectKey  client.ObjectKey
	ClusterObjectKey client.ObjectKey
	Client           client.Client
	InstanceName     string
	SpoolDirectory   string
	PGDataPath       string
	PGWALPath        string
	wal.UnimplementedWALServer
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
		path.Join(w.PGDataPath, CheckEmptyWalArchiveFile),
	)
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

// Restore implements the WALService interface
func (w WALServiceImplementation) Restore(
	ctx context.Context,
	request *wal.WALRestoreRequest,
) (*wal.WALRestoreResult, error) {
	contextLogger := log.FromContext(ctx)
	startTime := time.Now()

	var cluster *cnpgv1.Cluster

	if err := w.Client.Get(ctx, w.ClusterObjectKey, cluster); err != nil {
		return nil, err
	}

	var objectStore barmancloudv1.ObjectStore
	if err := w.Client.Get(ctx, w.BarmanObjectKey, &objectStore); err != nil {
		return nil, err
	}

	// TODO: build full paths
	walName := request.GetSourceWalName()
	destinationPath := request.GetDestinationFileName()

	barmanConfiguration := &objectStore.Spec.Configuration

	env := getRestoreCABundleEnv(barmanConfiguration)
	credentialsEnv, err := barmanCredentials.EnvSetBackupCloudCredentials(
		ctx,
		w.Client,
		objectStore.Namespace,
		&objectStore.Spec.Configuration,
		os.Environ(),
	)
	if err != nil {
		return nil, fmt.Errorf("while getting recover credentials: %w", err)
	}
	mergeEnv(env, credentialsEnv)

	options, err := barmanCommand.CloudWalRestoreOptions(ctx, barmanConfiguration, objectStore.Name)
	if err != nil {
		return nil, fmt.Errorf("while getting barman-cloud-wal-restore options: %w", err)
	}

	// Create the restorer
	var walRestorer *barmanRestorer.WALRestorer
	if walRestorer, err = barmanRestorer.New(ctx, env, w.SpoolDirectory); err != nil {
		return nil, fmt.Errorf("while creating the restorer: %w", err)
	}

	// Step 1: check if this WAL file is not already in the spool
	var wasInSpool bool
	if wasInSpool, err = walRestorer.RestoreFromSpool(walName, destinationPath); err != nil {
		return nil, fmt.Errorf("while restoring a file from the spool directory: %w", err)
	}
	if wasInSpool {
		contextLogger.Info("Restored WAL file from spool (parallel)",
			"walName", walName,
		)
		return nil, nil
	}

	// We skip this step if streaming connection is not available
	if isStreamingAvailable(cluster, w.InstanceName) {
		if err := checkEndOfWALStreamFlag(walRestorer); err != nil {
			return nil, err
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
			return nil, fmt.Errorf("while generating the list of WAL files to restore: %w", err)
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
		return nil, walStatus[0].Err
	}

	// We skip this step if streaming connection is not available
	endOfWALStream := isEndOfWALStream(walStatus)
	if isStreamingAvailable(cluster, w.InstanceName) && endOfWALStream {
		contextLogger.Info(
			"Set end-of-wal-stream flag as one of the WAL files to be prefetched was not found")

		err = walRestorer.SetEndOfWALStream()
		if err != nil {
			return nil, err
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

	return &wal.WALRestoreResult{}, nil
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

// mergeEnv merges all the values inside incomingEnv into env.
func mergeEnv(env []string, incomingEnv []string) {
	for _, incomingItem := range incomingEnv {
		incomingKV := strings.SplitAfterN(incomingItem, "=", 2)
		if len(incomingKV) != 2 {
			continue
		}
		for idx, item := range env {
			if strings.HasPrefix(item, incomingKV[0]) {
				env[idx] = incomingItem
			}
		}
	}
}

// TODO: refactor.
const (
	// ScratchDataDirectory is the directory to be used for scratch data.
	ScratchDataDirectory = "/controller"

	// CertificatesDir location to store the certificates.
	CertificatesDir = ScratchDataDirectory + "/certificates/"

	// BarmanBackupEndpointCACertificateLocation is the location where the barman endpoint
	// CA certificate is stored.
	BarmanBackupEndpointCACertificateLocation = CertificatesDir + BarmanBackupEndpointCACertificateFileName

	// BarmanBackupEndpointCACertificateFileName is the name of the file in which the barman endpoint
	// CA certificate for backups is stored.
	BarmanBackupEndpointCACertificateFileName = "backup-" + BarmanEndpointCACertificateFileName

	// BarmanRestoreEndpointCACertificateFileName is the name of the file in which the barman endpoint
	// CA certificate for restores is stored.
	BarmanRestoreEndpointCACertificateFileName = "restore-" + BarmanEndpointCACertificateFileName

	// BarmanEndpointCACertificateFileName is the name of the file in which the barman endpoint
	// CA certificate is stored.
	BarmanEndpointCACertificateFileName = "barman-ca.crt"
)

func getRestoreCABundleEnv(configuration *barmanapi.BarmanObjectStoreConfiguration) []string {
	var env []string

	if configuration.EndpointCA != nil && configuration.BarmanCredentials.AWS != nil {
		env = append(env, fmt.Sprintf("AWS_CA_BUNDLE=%s", BarmanBackupEndpointCACertificateLocation))
	} else if configuration.EndpointCA != nil && configuration.BarmanCredentials.Azure != nil {
		env = append(env, fmt.Sprintf("REQUESTS_CA_BUNDLE=%s", BarmanBackupEndpointCACertificateLocation))
	}
	return env
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
