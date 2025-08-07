package instance

import (
	"context"
	"fmt"
	"os"
	"slices"
	"time"

	barmanCommand "github.com/cloudnative-pg/barman-cloud/pkg/command"
	barmanCredentials "github.com/cloudnative-pg/barman-cloud/pkg/credentials"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/machinery/pkg/log"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/common"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"
)

// defaultRetentionPolicyInterval is the retention policy interval
// used when the current cluster or barman object store can't
// be read or when the enforcement process failed
const defaultRetentionPolicyInterval = time.Minute * 5

// CatalogMaintenanceRunnable executes all the barman catalog maintenance operations
type CatalogMaintenanceRunnable struct {
	Client         client.Client
	Recorder       record.EventRecorder
	ClusterKey     types.NamespacedName
	CurrentPodName string
}

// Start enforces the backup retention policies periodically, using the
// period specified in the BarmanObjectStore object
func (c *CatalogMaintenanceRunnable) Start(ctx context.Context) error {
	contextLogger := log.FromContext(ctx)
	contextLogger.Info("Starting retention policy runnable")

	for {
		// Enforce the retention policies
		period, err := c.cycle(ctx)
		if err != nil {
			contextLogger.Error(err, "Retention policy enforcement failed")
		}

		if period == 0 {
			period = defaultRetentionPolicyInterval
		}

		select {
		case <-time.After(period):
		case <-ctx.Done():
			return nil
		}
	}
}

// cycle enforces the retention policies. On success, it returns the amount
// of time to wait to the next check.
func (c *CatalogMaintenanceRunnable) cycle(ctx context.Context) (time.Duration, error) {
	var cluster cnpgv1.Cluster
	var barmanObjectStore barmancloudv1.ObjectStore

	if err := c.Client.Get(ctx, c.ClusterKey, &cluster); err != nil {
		return 0, err
	}

	configuration := config.NewFromCluster(&cluster)
	if err := c.Client.Get(ctx, configuration.GetBarmanObjectKey(), &barmanObjectStore); err != nil {
		return 0, err
	}

	if err := c.maintenance(ctx, &cluster, &barmanObjectStore); err != nil {
		return 0, err
	}

	nextCheckInterval := time.Second * time.Duration(
		barmanObjectStore.Spec.InstanceSidecarConfiguration.RetentionPolicyIntervalSeconds)
	return nextCheckInterval, nil
}

// maintenance executes a collection of operations:
//
// - applies the retention policy to the object.
//
// - store and deletes the stale Kubernetes backup objects.
//
// - updates the first recoverability point.
func (c *CatalogMaintenanceRunnable) maintenance(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	objectStore *barmancloudv1.ObjectStore,
) error {
	contextLogger := log.FromContext(ctx)
	configuration := config.NewFromCluster(cluster)
	retentionPolicy := objectStore.Spec.RetentionPolicy

	if cluster.Status.CurrentPrimary != c.CurrentPodName {
		contextLogger.Info(
			"Skipping retention policy enforcement, not the current primary",
			"currentPrimary", cluster.Status.CurrentPrimary, "podName", c.CurrentPodName)
		return nil
	}

	env, err := barmanCredentials.EnvSetCloudCredentialsAndCertificates(
		ctx,
		c.Client,
		objectStore.Namespace,
		&objectStore.Spec.Configuration,
		os.Environ(),
		common.BuildCertificateFilePath(objectStore.Name),
	)
	if err != nil {
		contextLogger.Error(err, "while setting backup cloud credentials")
		return err
	}

	if len(retentionPolicy) == 0 {
		contextLogger.Info("Skipping retention policy enforcement, no retention policy specified")
	} else {
		contextLogger.Info("Applying backup retention policy",
			"retentionPolicy", retentionPolicy)

		if err := barmanCommand.DeleteBackupsByPolicy(
			ctx,
			&objectStore.Spec.Configuration,
			configuration.ServerName,
			env,
			retentionPolicy,
		); err != nil {
			contextLogger.Error(err, "while enforcing retention policies")
			c.Recorder.Event(cluster, "Warning", "RetentionPolicyFailed", "Retention policy failed")
			return err
		}
	}

	backupList, err := barmanCommand.GetBackupList(
		ctx,
		&objectStore.Spec.Configuration,
		configuration.ServerName,
		env,
	)
	if err != nil {
		contextLogger.Error(err, "while reading the backup list")
		return err
	}

	if err := deleteBackupsNotInCatalog(ctx, c.Client, cluster, backupList.GetBackupIDs()); err != nil {
		contextLogger.Error(err, "while deleting Backups not present in the catalog")
		return err
	}

	return updateRecoveryWindow(ctx, c.Client, backupList, objectStore, configuration.ServerName)
}

// deleteBackupsNotInCatalog deletes all Backup objects pointing to the given cluster that are not
// present in the backup anymore
func deleteBackupsNotInCatalog(
	ctx context.Context,
	cli client.Client,
	cluster *cnpgv1.Cluster,
	backupIDs []string,
) error {
	// We had two options:
	//
	// A. quicker
	// get policy checker function
	// get all backups in the namespace for this cluster
	// check with policy checker function if backup should be deleted, then delete it if true
	//
	// B. more precise
	// get the catalog (GetBackupList)
	// get all backups in the namespace for this cluster
	// go through all backups and delete them if not in the catalog
	//
	// 1: all backups in the bucket should be also in the cluster
	// 2: all backups in the cluster should be in the bucket
	//
	// A can violate 1 and 2
	// A + B can still violate 2
	// B satisfies 1 and 2
	//
	// We chose to go with B

	contextLogger := log.FromContext(ctx)
	contextLogger.Debug("Checking the catalog to delete backups not present anymore")

	backups := cnpgv1.BackupList{}
	if err := cli.List(ctx, &backups, client.InNamespace(cluster.GetNamespace())); err != nil {
		return fmt.Errorf("while getting backups: %w", err)
	}

	var errors []error
	for id, backup := range backups.Items {
		if backup.Spec.Cluster.Name != cluster.GetName() ||
			backup.Status.Phase != cnpgv1.BackupPhaseCompleted ||
			!useSameBackupLocation(&backup.Status, cluster) {
			continue
		}

		// here we could add further checks, e.g. if the backup is not found but would still
		// be in the retention policy we could either not delete it or update it is status
		if !slices.Contains(backupIDs, backup.Status.BackupID) {
			contextLogger.Info("Deleting backup not in the catalog", "backup", backup.Name)
			if err := cli.Delete(ctx, &backups.Items[id]); err != nil {
				errors = append(errors, fmt.Errorf(
					"while deleting backup %s/%s: %w",
					backup.Namespace,
					backup.Name,
					err,
				))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("got errors while deleting Backups not in the cluster: %v", errors)
	}

	return nil
}

// useSameBackupLocation checks whether the given backup was taken using the same configuration as provided
func useSameBackupLocation(backup *cnpgv1.BackupStatus, cluster *cnpgv1.Cluster) bool {
	if backup.Method != cnpgv1.BackupMethodPlugin {
		return false
	}

	meta := newBackupResultMetadataFromMap(backup.PluginMetadata)
	return meta.clusterUID == string(cluster.UID) && meta.pluginName == metadata.PluginName
}
