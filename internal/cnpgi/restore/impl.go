package restore

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudnative-pg/barman-cloud/pkg/api"
	barmanArchiver "github.com/cloudnative-pg/barman-cloud/pkg/archiver"
	barmanCapabilities "github.com/cloudnative-pg/barman-cloud/pkg/capabilities"
	barmanCommand "github.com/cloudnative-pg/barman-cloud/pkg/command"
	barmanCredentials "github.com/cloudnative-pg/barman-cloud/pkg/credentials"
	barmanRestorer "github.com/cloudnative-pg/barman-cloud/pkg/restorer"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/utils"
	"github.com/cloudnative-pg/machinery/pkg/execlog"
	"github.com/cloudnative-pg/machinery/pkg/fileutils"
	"github.com/cloudnative-pg/machinery/pkg/log"
	"os"
	"os/exec"
	"path"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ScratchDataDirectory is the directory to be used for scratch data
	ScratchDataDirectory = "/controller"

	// RecoveryTemporaryDirectory provides a path to store temporary files
	// needed in the recovery process
	RecoveryTemporaryDirectory = ScratchDataDirectory + "/recovery"
)

type JobHookImpl struct {
	Cli                 client.Client
	Namespace           string
	SpoolDirectory      string
	EmptyWALArchiveFile string
	PgData              string
	PgWal               string
}

type restoreDataReq struct {
	ClusterName string
	BackupName  string
}

type RestoreDataRes struct {
}

func (impl JobHookImpl) RestoreDirectories(ctx context.Context, req restoreDataReq) (*RestoreDataRes, error) {
	var cluster cnpgv1.Cluster
	if err := impl.Cli.Get(ctx, client.ObjectKey{Namespace: impl.Namespace, Name: req.ClusterName}, &cluster); err != nil {
		return nil, err
	}

	// Before starting the restore we check if the archive destination is safe to use
	// otherwise, we stop creating the cluster
	if err := impl.checkBackupDestination(ctx, &cluster); err != nil {
		return nil, err
	}

	// If we need to download data from a backup, we do it
	backup, env, err := impl.loadBackup(ctx, req.BackupName)
	if err != nil {
		return nil, err
	}

	if err := impl.ensureArchiveContainsLastCheckpointRedoWAL(ctx, &cluster, env, backup); err != nil {
		return nil, err
	}

	if err := impl.restoreDataDir(ctx, backup, env); err != nil {
		return nil, err
	}

	if _, err := impl.restoreCustomWalDir(ctx); err != nil {
		return nil, err
	}

	return &RestoreDataRes{}, nil
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

	options, err := barmanCommand.AppendCloudProviderOptionsFromBackup(ctx, options, backup.Status.BarmanCredentials)
	if err != nil {
		return err
	}

	options = append(options, impl.PgData)

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

	opts, err := barmanCommand.CloudWalRestoreOptions(ctx, &api.BarmanObjectStoreConfiguration{
		BarmanCredentials: backup.Status.BarmanCredentials,
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
	if !cluster.Spec.Backup.IsBarmanBackupConfigured() {
		return nil
	}

	// Get environment from cache
	env, err := barmanCredentials.EnvSetRestoreCloudCredentials(ctx,
		impl.Cli,
		cluster.Namespace,
		cluster.Spec.Backup.BarmanObjectStore,
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
		impl.PgData,
		path.Join(impl.PgData, impl.EmptyWALArchiveFile))
	if err != nil {
		return fmt.Errorf("while creating the archiver: %w", err)
	}

	// Get WAL archive options
	checkWalOptions, err := walArchiver.BarmanCloudCheckWalArchiveOptions(
		ctx, cluster.Spec.Backup.BarmanObjectStore, cluster.Name)
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

func (impl JobHookImpl) loadBackup(
	ctx context.Context,
	backupName string,
) (*cnpgv1.Backup, []string, error) {
	var backup cnpgv1.Backup
	err := impl.Cli.Get(
		ctx,
		client.ObjectKey{Namespace: impl.Namespace, Name: backupName},
		&backup)
	if err != nil {
		return nil, nil, err
	}

	env, err := barmanCredentials.EnvSetRestoreCloudCredentials(
		ctx,
		impl.Cli,
		impl.Namespace,
		&api.BarmanObjectStoreConfiguration{
			BarmanCredentials: backup.Status.BarmanCredentials,
			EndpointCA:        backup.Status.EndpointCA,
			EndpointURL:       backup.Status.EndpointURL,
			DestinationPath:   backup.Status.DestinationPath,
			ServerName:        backup.Status.ServerName,
		},
		os.Environ())
	if err != nil {
		return nil, nil, err
	}

	log.Info("Recovering existing backup", "backup", backup)
	return &backup, env, nil
}

// restoreCustomWalDir moves the current pg_wal data to the specified custom wal dir and applies the symlink
// returns indicating if any changes were made and any error encountered in the process
func (impl JobHookImpl) restoreCustomWalDir(ctx context.Context) (bool, error) {
	const pgWalDirectory = "pg_wal"

	if impl.PgWal == "" {
		return false, nil
	}

	contextLogger := log.FromContext(ctx)
	pgDataWal := path.Join(impl.PgData, pgWalDirectory)

	// if the link is already present we have nothing to do.
	if linkInfo, _ := os.Readlink(pgDataWal); linkInfo == impl.PgWal {
		contextLogger.Info("symlink to the WAL volume already present, skipping the custom wal dir restore")
		return false, nil
	}

	if err := fileutils.EnsureDirectoryExists(impl.PgWal); err != nil {
		return false, err
	}

	contextLogger.Info("restoring WAL volume symlink and transferring data")
	if err := fileutils.EnsureDirectoryExists(pgDataWal); err != nil {
		return false, err
	}

	if err := fileutils.MoveDirectoryContent(pgDataWal, impl.PgWal); err != nil {
		return false, err
	}

	if err := fileutils.RemoveFile(pgDataWal); err != nil {
		return false, err
	}

	return true, os.Symlink(impl.PgWal, pgDataWal)
}
