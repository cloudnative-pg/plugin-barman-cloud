package instance

import (
	"context"
	"fmt"
	"os"
	"time"

	barmanBackup "github.com/cloudnative-pg/barman-cloud/pkg/backup"
	barmanCommand "github.com/cloudnative-pg/barman-cloud/pkg/command"
	barmanCredentials "github.com/cloudnative-pg/barman-cloud/pkg/credentials"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/postgres"
	"github.com/cloudnative-pg/cnpg-i/pkg/backup"
	"github.com/cloudnative-pg/machinery/pkg/fileutils"
	"github.com/cloudnative-pg/machinery/pkg/log"
	pgTime "github.com/cloudnative-pg/machinery/pkg/postgres/time"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/common"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"
)

// BackupServiceImplementation is the implementation
// of the Backup CNPG capability
type BackupServiceImplementation struct {
	Client       client.Client
	InstanceName string
	backup.UnimplementedBackupServer
}

// GetCapabilities implements the BackupService interface
func (b BackupServiceImplementation) GetCapabilities(
	_ context.Context, _ *backup.BackupCapabilitiesRequest,
) (*backup.BackupCapabilitiesResult, error) {
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

// Backup implements the Backup interface
func (b BackupServiceImplementation) Backup(
	ctx context.Context,
	request *backup.BackupRequest,
) (*backup.BackupResult, error) {
	contextLogger := log.FromContext(ctx)

	contextLogger.Info("Starting backup")

	configuration, err := config.NewFromClusterJSON(request.ClusterDefinition)
	if err != nil {
		return nil, err
	}

	var objectStore barmancloudv1.ObjectStore
	if err := b.Client.Get(ctx, configuration.GetBarmanObjectKey(), &objectStore); err != nil {
		contextLogger.Error(err, "while getting object store", "key", configuration.GetRecoveryBarmanObjectKey())
		return nil, err
	}

	if err := fileutils.EnsureDirectoryExists(postgres.BackupTemporaryDirectory); err != nil {
		contextLogger.Error(err, "Cannot create backup temporary directory", "err", err)
		return nil, err
	}

	backupCmd := barmanBackup.NewBackupCommand(&objectStore.Spec.Configuration)

	// We need to connect to PostgreSQL and to do that we need
	// PGHOST (and the like) to be available
	osEnvironment := os.Environ()
	caBundleEnvironment := common.GetRestoreCABundleEnv(&objectStore.Spec.Configuration)
	env, err := barmanCredentials.EnvSetCloudCredentialsAndCertificates(
		ctx,
		b.Client,
		objectStore.Namespace,
		&objectStore.Spec.Configuration,
		common.MergeEnv(osEnvironment, caBundleEnvironment),
		common.BuildCertificateFilePath(objectStore.Name),
	)
	if err != nil {
		contextLogger.Error(err, "while setting backup cloud credentials")
		return nil, err
	}

	backupName := fmt.Sprintf("backup-%v", pgTime.ToCompactISO8601(time.Now()))

	if err = backupCmd.Take(
		ctx,
		backupName,
		configuration.ServerName,
		env,
		postgres.BackupTemporaryDirectory,
	); err != nil {
		contextLogger.Error(err, "while taking backup")
		return nil, err
	}

	executedBackupInfo, err := backupCmd.GetExecutedBackupInfo(
		ctx,
		backupName,
		configuration.ServerName,
		env)
	if err != nil {
		contextLogger.Error(err, "while getting executed backup info")
		return nil, err
	}

	contextLogger.Info("Backup completed", "backup", executedBackupInfo.ID)

	// Refresh the recovery window
	contextLogger.Info(
		"Refreshing the recovery window",
		"backupName", executedBackupInfo.BackupName,
	)
	backupList, err := barmanCommand.GetBackupList(
		ctx,
		&objectStore.Spec.Configuration,
		configuration.ServerName,
		env,
	)
	if err != nil {
		contextLogger.Error(err, "while reading the backup list")
		return nil, err
	}

	if err := updateRecoveryWindow(
		ctx,
		b.Client,
		backupList,
		&objectStore,
		configuration.ServerName,
	); err != nil {
		contextLogger.Error(
			err,
			"Error while updating the recovery window in the ObjectStore status stanza. Skipping.",
			"backupName", executedBackupInfo.BackupName,
		)
	} else {
		contextLogger.Debug(
			"backupName", executedBackupInfo.BackupName,
			"Updated the recovery window in the ObjectStore status stanza",
			"serverRecoveryWindow", objectStore.Status.ServerRecoveryWindow,
		)
	}

	return &backup.BackupResult{
		BackupId:   executedBackupInfo.ID,
		BackupName: executedBackupInfo.BackupName,
		StartedAt:  metav1.Time{Time: executedBackupInfo.BeginTime}.Unix(),
		StoppedAt:  metav1.Time{Time: executedBackupInfo.EndTime}.Unix(),
		BeginWal:   executedBackupInfo.BeginWal,
		EndWal:     executedBackupInfo.EndWal,
		BeginLsn:   executedBackupInfo.BeginLSN,
		EndLsn:     executedBackupInfo.EndLSN,
		InstanceId: b.InstanceName,
		Online:     true,
		Metadata:   newBackupResultMetadata(configuration.Cluster.ObjectMeta.UID, executedBackupInfo.TimeLine).toMap(),
	}, nil
}
