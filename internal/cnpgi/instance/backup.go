package instance

import (
	"context"
	"os"

	barmanBackup "github.com/cloudnative-pg/barman-cloud/pkg/backup"
	barmanCapabilities "github.com/cloudnative-pg/barman-cloud/pkg/capabilities"
	barmanCredentials "github.com/cloudnative-pg/barman-cloud/pkg/credentials"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/postgres"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/decoder"
	"github.com/cloudnative-pg/cnpg-i/pkg/backup"
	"github.com/cloudnative-pg/machinery/pkg/fileutils"
	"github.com/cloudnative-pg/machinery/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BackupServiceImplementation is the implementation
// of the Backup CNPG capability
type BackupServiceImplementation struct {
	Client       client.Client
	Recorder     record.EventRecorder
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
	req *backup.BackupRequest,
) (*backup.BackupResult, error) {
	contextLogger := log.FromContext(ctx)
	backupObj, err := decoder.DecodeBackup(req.BackupDefinition)
	if err != nil {
		return nil, err
	}
	cluster, err := decoder.DecodeClusterJSON(req.ClusterDefinition)
	if err != nil {
		return nil, err
	}

	if err := fileutils.EnsureDirectoryExists(postgres.BackupTemporaryDirectory); err != nil {
		contextLogger.Error(err, "Cannot create backup temporary directory", "err", err)
		return nil, err
	}

	capabilities, err := barmanCapabilities.CurrentCapabilities()
	if err != nil {
		return nil, err
	}
	backupCmd := barmanBackup.NewBackupCommand(
		cluster.Spec.Backup.BarmanObjectStore,
		capabilities,
	)
	env := os.Environ()
	env, err = barmanCredentials.EnvSetBackupCloudCredentials(
		ctx,
		b.Client,
		cluster.Namespace,
		cluster.Spec.Backup.BarmanObjectStore,
		env)
	if err != nil {
		return nil, err
	}

	if err = backupCmd.Take(
		ctx,
		backupObj.Status.BackupName,
		backupObj.Status.ServerName,
		env,
		cluster,
		postgres.BackupTemporaryDirectory,
	); err != nil {
		return nil, err
	}

	contextLogger.Info("Backup completed")
	b.Recorder.Event(backupObj, "Normal", "Completed", "Backup completed")

	// Set the status to completed
	backupObj.Status.SetAsCompleted()

	executedBackupInfo, err := backupCmd.GetExecutedBackupInfo(
		ctx, backupObj.Status.BackupName, backupObj.Status.ServerName, cluster, env)
	if err != nil {
		return nil, err
	}

	return &backup.BackupResult{
		BackupId:          executedBackupInfo.ID,
		BackupName:        executedBackupInfo.BackupName,
		StartedAt:         metav1.Time{Time: executedBackupInfo.BeginTime}.Unix(),
		StoppedAt:         metav1.Time{Time: executedBackupInfo.EndTime}.Unix(),
		BeginWal:          executedBackupInfo.BeginWal,
		EndWal:            executedBackupInfo.EndWal,
		BeginLsn:          executedBackupInfo.BeginLSN,
		EndLsn:            executedBackupInfo.EndLSN,
		BackupLabelFile:   nil,
		TablespaceMapFile: nil,
		InstanceId:        b.InstanceName,
		Online:            true,
		Metadata:          nil,
	}, nil
}
