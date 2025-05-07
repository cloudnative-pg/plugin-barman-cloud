package restore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/cloudnative-pg/barman-cloud/pkg/api"
	barmanArchiver "github.com/cloudnative-pg/barman-cloud/pkg/archiver"
	barmanCatalog "github.com/cloudnative-pg/barman-cloud/pkg/catalog"
	barmanCommand "github.com/cloudnative-pg/barman-cloud/pkg/command"
	barmanCredentials "github.com/cloudnative-pg/barman-cloud/pkg/credentials"
	barmanRestorer "github.com/cloudnative-pg/barman-cloud/pkg/restorer"
	barmanUtils "github.com/cloudnative-pg/barman-cloud/pkg/utils"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/postgres"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/utils"
	restore "github.com/cloudnative-pg/cnpg-i/pkg/restore/job"
	"github.com/cloudnative-pg/machinery/pkg/execlog"
	"github.com/cloudnative-pg/machinery/pkg/fileutils"
	"github.com/cloudnative-pg/machinery/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/common"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"
)

const (
	// ScratchDataDirectory is the directory to be used for scratch data
	ScratchDataDirectory = "/controller"

	// RecoveryTemporaryDirectory provides a path to store temporary files
	// needed in the recovery process
	RecoveryTemporaryDirectory = ScratchDataDirectory + "/recovery"
)

// JobHookImpl is the implementation of the restore job hooks
type JobHookImpl struct {
	restore.UnimplementedRestoreJobHooksServer

	Client client.Client

	SpoolDirectory       string
	PgDataPath           string
	PgWalFolderToSymlink string
}

// GetCapabilities returns the capabilities of the restore job hooks
func (impl JobHookImpl) GetCapabilities(
	_ context.Context,
	_ *restore.RestoreJobHooksCapabilitiesRequest,
) (*restore.RestoreJobHooksCapabilitiesResult, error) {
	return &restore.RestoreJobHooksCapabilitiesResult{
		Capabilities: []*restore.RestoreJobHooksCapability{
			{
				Kind: restore.RestoreJobHooksCapability_KIND_RESTORE,
			},
		},
	}, nil
}

// Restore restores the cluster from a backup
func (impl JobHookImpl) Restore(
	ctx context.Context,
	req *restore.RestoreRequest,
) (*restore.RestoreResponse, error) {
	contextLogger := log.FromContext(ctx)

	configuration, err := config.NewFromClusterJSON(req.ClusterDefinition)
	if err != nil {
		return nil, err
	}

	var recoveryObjectStore barmancloudv1.ObjectStore
	if err := impl.Client.Get(ctx, configuration.GetRecoveryBarmanObjectKey(), &recoveryObjectStore); err != nil {
		return nil, err
	}

	if configuration.BarmanObjectName != "" {
		var targetObjectStore barmancloudv1.ObjectStore
		if err := impl.Client.Get(ctx, configuration.GetBarmanObjectKey(), &targetObjectStore); err != nil {
			return nil, err
		}

		if err := impl.checkBackupDestination(
			ctx,
			configuration.Cluster,
			&targetObjectStore.Spec.Configuration,
			targetObjectStore.Name,
		); err != nil {
			return nil, err
		}
	}

	// Detect the backup to recover
	backup, env, err := loadBackupObjectFromExternalCluster(
		ctx,
		impl.Client,
		configuration.Cluster,
		&recoveryObjectStore.Spec.Configuration,
		recoveryObjectStore.Name,
		configuration.RecoveryServerName,
	)
	if err != nil {
		return nil, err
	}

	if err := impl.ensureArchiveContainsLastCheckpointRedoWAL(
		ctx,
		env,
		backup,
		&recoveryObjectStore.Spec.Configuration,
	); err != nil {
		return nil, err
	}

	if err := impl.restoreDataDir(
		ctx,
		backup,
		env,
		&recoveryObjectStore.Spec.Configuration,
	); err != nil {
		return nil, err
	}

	if configuration.Cluster.Spec.WalStorage != nil {
		if _, err := impl.restoreCustomWalDir(ctx); err != nil {
			return nil, err
		}
	}

	config := getRestoreWalConfig()

	contextLogger.Info("sending restore response", "config", config, "env", env)
	return &restore.RestoreResponse{
		RestoreConfig: config,
		Envs:          nil,
	}, nil
}

// restoreDataDir restores PGDATA from an existing backup
func (impl JobHookImpl) restoreDataDir(
	ctx context.Context,
	backup *cnpgv1.Backup,
	env []string,
	barmanConfiguration *cnpgv1.BarmanObjectStoreConfiguration,
) error {
	var options []string

	options, err := barmanCommand.AppendCloudProviderOptionsFromConfiguration(ctx, options, barmanConfiguration)
	if err != nil {
		return err
	}

	if backup.Status.EndpointURL != "" {
		options = append(options, "--endpoint-url", backup.Status.EndpointURL)
	}
	options = append(options, backup.Status.DestinationPath)
	options = append(options, backup.Status.ServerName)
	options = append(options, backup.Status.BackupID)
	options = append(options, impl.PgDataPath)

	log.Info("Starting barman-cloud-restore",
		"options", options)

	cmd := exec.Command(barmanUtils.BarmanCloudRestore, options...) // #nosec G204
	cmd.Env = env
	err = execlog.RunStreaming(cmd, barmanUtils.BarmanCloudRestore)
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			err = barmanCommand.UnmarshalBarmanCloudRestoreExitCode(exitError.ExitCode())
		}

		log.Error(err, "Can't restore backup")
		return err
	}
	log.Info("Restore completed")
	return nil
}

func (impl JobHookImpl) ensureArchiveContainsLastCheckpointRedoWAL(
	ctx context.Context,
	env []string,
	backup *cnpgv1.Backup,
	barmanConfiguration *cnpgv1.BarmanObjectStoreConfiguration,
) error {
	// it's the full path of the file that will temporarily contain the LastCheckpointRedoWAL
	const testWALPath = RecoveryTemporaryDirectory + "/test.wal"
	contextLogger := log.FromContext(ctx)

	defer func() {
		if err := fileutils.RemoveFile(testWALPath); err != nil {
			contextLogger.Error(err, "while deleting the temporary wal file: %w")
		}
	}()

	if err := fileutils.EnsureParentDirectoryExists(testWALPath); err != nil {
		return err
	}

	rest, err := barmanRestorer.New(ctx, env, impl.SpoolDirectory)
	if err != nil {
		return err
	}

	opts, err := barmanCommand.CloudWalRestoreOptions(ctx, barmanConfiguration, backup.Status.ServerName)
	if err != nil {
		return err
	}

	if err := rest.Restore(backup.Status.BeginWal, testWALPath, opts); err != nil {
		return fmt.Errorf("encountered an error while checking the presence of first needed WAL in the archive: %w", err)
	}

	return nil
}

func (impl *JobHookImpl) checkBackupDestination(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	barmanConfiguration *cnpgv1.BarmanObjectStoreConfiguration,
	objectStoreName string,
) error {
	// Get environment from cache
	env, err := barmanCredentials.EnvSetCloudCredentialsAndCertificates(ctx,
		impl.Client,
		cluster.Namespace,
		barmanConfiguration,
		os.Environ(),
		common.BuildCertificateFilePath(objectStoreName),
	)
	if err != nil {
		return fmt.Errorf("can't get credentials for cluster %v: %w", cluster.Name, err)
	}
	if len(env) == 0 {
		return nil
	}

	// Instantiate the WALArchiver to get the proper configuration
	var walArchiver *barmanArchiver.WALArchiver
	walArchiver, err = barmanArchiver.New(
		ctx,
		env,
		impl.SpoolDirectory,
		impl.PgDataPath,
		path.Join(impl.PgDataPath, metadata.CheckEmptyWalArchiveFile))
	if err != nil {
		return fmt.Errorf("while creating the archiver: %w", err)
	}

	// TODO: refactor this code elsewhere
	serverName := cluster.Name
	for _, plugin := range cluster.Spec.Plugins {
		if plugin.IsEnabled() && plugin.Name == metadata.PluginName {
			if pluginServerName, ok := plugin.Parameters["serverName"]; ok {
				serverName = pluginServerName
			}
		}
	}

	// Get WAL archive options
	checkWalOptions, err := walArchiver.BarmanCloudCheckWalArchiveOptions(
		ctx, barmanConfiguration, serverName)
	if err != nil {
		log.Error(err, "while getting barman-cloud-wal-archive options")
		return err
	}

	// Check if we're ok to archive in the desired destination
	if utils.IsEmptyWalArchiveCheckEnabled(&cluster.ObjectMeta) {
		return walArchiver.CheckWalArchiveDestination(ctx, checkWalOptions)
	}

	return nil
}

// restoreCustomWalDir moves the current pg_wal data to the specified custom wal dir and applies the symlink
// returns indicating if any changes were made and any error encountered in the process
func (impl JobHookImpl) restoreCustomWalDir(ctx context.Context) (bool, error) {
	const pgWalDirectory = "pg_wal"

	contextLogger := log.FromContext(ctx)
	pgDataWal := path.Join(impl.PgDataPath, pgWalDirectory)

	// if the link is already present we have nothing to do.
	if linkInfo, _ := os.Readlink(pgDataWal); linkInfo == impl.PgWalFolderToSymlink {
		contextLogger.Info("symlink to the WAL volume already present, skipping the custom wal dir restore")
		return false, nil
	}

	if err := fileutils.EnsureDirectoryExists(impl.PgWalFolderToSymlink); err != nil {
		return false, err
	}

	contextLogger.Info("restoring WAL volume symlink and transferring data")
	if err := fileutils.EnsureDirectoryExists(pgDataWal); err != nil {
		return false, err
	}

	if err := fileutils.MoveDirectoryContent(pgDataWal, impl.PgWalFolderToSymlink); err != nil {
		return false, err
	}

	if err := fileutils.RemoveFile(pgDataWal); err != nil {
		return false, err
	}

	return true, os.Symlink(impl.PgWalFolderToSymlink, pgDataWal)
}

// getRestoreWalConfig obtains the content to append to `custom.conf` allowing PostgreSQL
// to complete the WAL recovery from the object storage and then start
// as a new primary
func getRestoreWalConfig() string {
	restoreCmd := fmt.Sprintf(
		"/controller/manager wal-restore --log-destination %s/%s.json %%f %%p",
		postgres.LogPath, postgres.LogFileName)

	recoveryFileContents := fmt.Sprintf(
		"recovery_target_action = promote\n"+
			"restore_command = '%s'\n",
		restoreCmd)

	return recoveryFileContents
}

// loadBackupObjectFromExternalCluster generates an in-memory Backup structure given a reference to
// an external cluster, loading the required information from the object store
func loadBackupObjectFromExternalCluster(
	ctx context.Context,
	typedClient client.Client,
	cluster *cnpgv1.Cluster,
	recoveryObjectStore *api.BarmanObjectStoreConfiguration,
	recoveryObjectStoreName string,
	serverName string,
) (*cnpgv1.Backup, []string, error) {
	contextLogger := log.FromContext(ctx)

	contextLogger.Info("Recovering from external cluster",
		"serverName", serverName,
		"objectStore", recoveryObjectStore)

	env, err := barmanCredentials.EnvSetCloudCredentialsAndCertificates(
		ctx,
		typedClient,
		cluster.Namespace,
		recoveryObjectStore,
		os.Environ(),
		common.BuildCertificateFilePath(recoveryObjectStoreName))
	if err != nil {
		return nil, nil, err
	}

	contextLogger.Info("Downloading backup catalog")
	backupCatalog, err := barmanCommand.GetBackupList(ctx, recoveryObjectStore, serverName, env)
	if err != nil {
		return nil, nil, err
	}
	contextLogger.Info("Downloaded backup catalog", "backupCatalog", backupCatalog)

	// We are now choosing the right backup to restore
	var targetBackup *barmanCatalog.BarmanBackup
	if cluster.Spec.Bootstrap.Recovery != nil &&
		cluster.Spec.Bootstrap.Recovery.RecoveryTarget != nil {
		targetBackup, err = backupCatalog.FindBackupInfo(
			cluster.Spec.Bootstrap.Recovery.RecoveryTarget,
		)
		if err != nil {
			return nil, nil, err
		}
	} else {
		targetBackup = backupCatalog.LatestBackupInfo()
	}
	if targetBackup == nil {
		return nil, nil, fmt.Errorf("no target backup found")
	}

	contextLogger.Info("Target backup found", "backup", targetBackup)

	return &cnpgv1.Backup{
		Spec: cnpgv1.BackupSpec{
			Cluster: cnpgv1.LocalObjectReference{
				Name: serverName,
			},
		},
		Status: cnpgv1.BackupStatus{
			BarmanCredentials: recoveryObjectStore.BarmanCredentials,
			EndpointCA:        recoveryObjectStore.EndpointCA,
			EndpointURL:       recoveryObjectStore.EndpointURL,
			DestinationPath:   recoveryObjectStore.DestinationPath,
			ServerName:        serverName,
			BackupID:          targetBackup.ID,
			Phase:             cnpgv1.BackupPhaseCompleted,
			StartedAt:         &metav1.Time{Time: targetBackup.BeginTime},
			StoppedAt:         &metav1.Time{Time: targetBackup.EndTime},
			BeginWal:          targetBackup.BeginWal,
			EndWal:            targetBackup.EndWal,
			BeginLSN:          targetBackup.BeginLSN,
			EndLSN:            targetBackup.EndLSN,
			Error:             targetBackup.Error,
			CommandOutput:     "",
			CommandError:      "",
		},
	}, env, nil
}
