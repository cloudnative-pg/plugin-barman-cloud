package restore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/cloudnative-pg/barman-cloud/pkg/api"
	barmanArchiver "github.com/cloudnative-pg/barman-cloud/pkg/archiver"
	barmanCapabilities "github.com/cloudnative-pg/barman-cloud/pkg/capabilities"
	barmanCommand "github.com/cloudnative-pg/barman-cloud/pkg/command"
	barmanCredentials "github.com/cloudnative-pg/barman-cloud/pkg/credentials"
	barmanRestorer "github.com/cloudnative-pg/barman-cloud/pkg/restorer"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/utils"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/decoder"
	restore "github.com/cloudnative-pg/cnpg-i/pkg/restore/job"
	"github.com/cloudnative-pg/machinery/pkg/execlog"
	"github.com/cloudnative-pg/machinery/pkg/fileutils"
	"github.com/cloudnative-pg/machinery/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/client"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/common"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
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
	Client               client.Client
	ClusterObjectKey     client.ObjectKey
	BackupToRestore      client.ObjectKey
	ArchiveConfiguration client.ObjectKey
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
	var cluster cnpgv1.Cluster
	if err := decoder.DecodeObject(
		req.GetClusterDefinition(),
		&cluster,
		cnpgv1.GroupVersion.WithKind("Cluster"),
	); err != nil {
		return nil, err
	}
	// Before starting the restore we check if the archive destination is safe to use
	// otherwise, we stop creating the cluster
	if err := impl.checkBackupDestination(ctx, &cluster); err != nil {
		return nil, err
	}

	var backup cnpgv1.Backup

	if err := decoder.DecodeObject(
		req.GetBackupDefinition(),
		&backup,
		cnpgv1.GroupVersion.WithKind("Backup"),
	); err != nil {
		return nil, err
	}

	env, err := impl.getBarmanEnvFromBackup(ctx, &backup)
	if err != nil {
		return nil, err
	}

	if err := impl.ensureArchiveContainsLastCheckpointRedoWAL(ctx, &cluster, env, &backup); err != nil {
		return nil, err
	}

	if err := impl.restoreDataDir(ctx, &backup, env); err != nil {
		return nil, err
	}

	if cluster.Spec.WalStorage != nil {
		if _, err := impl.restoreCustomWalDir(ctx); err != nil {
			return nil, err
		}
	}

	config, err := getRestoreWalConfig(ctx, &backup)
	if err != nil {
		return nil, err
	}

	contextLogger.Info("sending restore response", "config", config, "env", env)
	return &restore.RestoreResponse{
		RestoreConfig: config,
		Envs:          env,
	}, nil
}

// restoreDataDir restores PGDATA from an existing backup
func (impl JobHookImpl) restoreDataDir(ctx context.Context, backup *cnpgv1.Backup, env []string) error {
	var options []string

	if backup.Status.EndpointURL != "" {
		options = append(options, "--endpoint-url", backup.Status.EndpointURL)
	}
	options = append(options, backup.Status.DestinationPath)
	options = append(options, backup.Status.ServerName)
	options = append(options, backup.Status.BackupID)

	creds, err := common.GetCredentialsFromBackup(backup)
	if err != nil {
		return err
	}
	options, err = barmanCommand.AppendCloudProviderOptionsFromBackup(ctx, options, creds)
	if err != nil {
		return err
	}

	options = append(options, impl.PgDataPath)

	log.Info("Starting barman-cloud-restore",
		"options", options)

	cmd := exec.Command(barmanCapabilities.BarmanCloudRestore, options...) // #nosec G204
	cmd.Env = env
	err = execlog.RunStreaming(cmd, barmanCapabilities.BarmanCloudRestore)
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			err = barmanCommand.UnmarshalBarmanCloudRestoreExitCode(ctx, exitError.ExitCode())
		}

		log.Error(err, "Can't restore backup")
		return err
	}
	log.Info("Restore completed")
	return nil
}

func (impl JobHookImpl) ensureArchiveContainsLastCheckpointRedoWAL(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	env []string,
	backup *cnpgv1.Backup,
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

	creds, err := common.GetCredentialsFromBackup(backup)
	if err != nil {
		return err
	}
	opts, err := barmanCommand.CloudWalRestoreOptions(ctx, &api.BarmanObjectStoreConfiguration{
		BarmanCredentials: creds,
		EndpointCA:        backup.Status.EndpointCA,
		EndpointURL:       backup.Status.EndpointURL,
		DestinationPath:   backup.Status.DestinationPath,
		ServerName:        backup.Status.ServerName,
	}, cluster.Name)
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
) error {
	if impl.ArchiveConfiguration.Name == "" {
		return nil
	}

	var barmanObj barmancloudv1.ObjectStore
	if err := impl.Client.Get(ctx, impl.ArchiveConfiguration, &barmanObj); err != nil {
		return err
	}

	// Get environment from cache
	env, err := barmanCredentials.EnvSetRestoreCloudCredentials(ctx,
		impl.Client,
		barmanObj.Namespace,
		&barmanObj.Spec.Configuration,
		os.Environ())
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

	// Get WAL archive options
	checkWalOptions, err := walArchiver.BarmanCloudCheckWalArchiveOptions(
		ctx, &barmanObj.Spec.Configuration, barmanObj.Name)
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

func (impl JobHookImpl) getBarmanEnvFromBackup(
	ctx context.Context,
	backup *cnpgv1.Backup,
) ([]string, error) {
	creds, err := common.GetCredentialsFromBackup(backup)
	if err != nil {
		return nil, err
	}
	env, err := barmanCredentials.EnvSetRestoreCloudCredentials(
		ctx,
		impl.Client,
		impl.BackupToRestore.Namespace,
		&api.BarmanObjectStoreConfiguration{
			BarmanCredentials: creds,
			EndpointURL:       backup.Status.EndpointURL,
			EndpointCA:        backup.Status.EndpointCA,
			DestinationPath:   backup.Status.DestinationPath,
			ServerName:        backup.Status.ServerName,
		},
		os.Environ())
	if err != nil {
		return nil, err
	}

	log.Info("Recovering existing backup", "backup", backup)
	return env, nil
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
func getRestoreWalConfig(ctx context.Context, backup *cnpgv1.Backup) (string, error) {
	var err error

	cmd := []string{barmanCapabilities.BarmanCloudWalRestore}
	if backup.Status.EndpointURL != "" {
		cmd = append(cmd, "--endpoint-url", backup.Status.EndpointURL)
	}
	cmd = append(cmd, backup.Status.DestinationPath)
	cmd = append(cmd, backup.Status.ServerName)

	creds, err := common.GetCredentialsFromBackup(backup)
	if err != nil {
		return "", err
	}

	cmd, err = barmanCommand.AppendCloudProviderOptionsFromBackup(ctx, cmd, creds)
	if err != nil {
		return "", err
	}

	cmd = append(cmd, "%f", "%p")

	recoveryFileContents := fmt.Sprintf(
		"recovery_target_action = promote\n"+
			"restore_command = '%s'\n",
		strings.Join(cmd, " "))

	return recoveryFileContents, nil
}
