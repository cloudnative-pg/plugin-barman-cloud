/*
Copyright © contributors to CloudNativePG, established as
CloudNativePG a Series of LF Projects, LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

SPDX-License-Identifier: Apache-2.0
*/

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
	contextLogger := log.FromContext(ctx)

	var cluster cnpgv1.Cluster
	var barmanObjectStore barmancloudv1.ObjectStore

	if err := c.Client.Get(ctx, c.ClusterKey, &cluster); err != nil {
		return 0, err
	}

	enabledPlugins := cnpgv1.GetPluginConfigurationEnabledPluginNames(cluster.Spec.Plugins)
	if !slices.Contains(enabledPlugins, metadata.PluginName) {
		contextLogger.Debug("Skipping maintenance cycle: plugin is not enabled for backups")
		return 0, nil
	}

	configuration := config.NewFromCluster(&cluster)
	if configuration == nil || len(configuration.BarmanObjectName) == 0 {
		return 0, fmt.Errorf("invalid configuration, missing barman object store reference")
	}

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
// present in the backup anymore.
//
// The deletion is guarded against short catalog listings (see #1115):
// barman-cloud-backup-list exits successfully even when some backup.info
// files cannot be read, so a transient object-store error can make the
// listing come back short (or empty) and every missing Backup object would
// be deleted. To avoid that, this function:
//
//   - skips the deletion pass entirely when the catalog is empty, and
//   - only deletes a Backup object when its backup is older than the oldest
//     entry still in the catalog. Those are the backups the retention policy
//     has already pruned from the bucket, so the catalog is expected to have
//     dropped them. A backup newer than the oldest catalog entry is still
//     inside the retention window and must still be listed; if it is missing,
//     the listing is short and the object is kept for the next cycle.
//
// A Backup object whose backup ID cannot be parsed is never deleted: when
// the age of a backup cannot be established, keeping it is the safe choice.
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

	if len(backupIDs) == 0 {
		// An empty catalog most likely means the listing failed or came back
		// short, not that the bucket is genuinely empty. Deleting every
		// completed Backup object on an empty listing would remove valid
		// backups, so skip this pass and let the next cycle retry.
		contextLogger.Info("Skipping deletion of backups not in the catalog: empty backup catalog")
		return nil
	}

	oldestCatalogBackup, err := oldestBackupTime(backupIDs)
	if err != nil {
		// Without a usable catalog boundary the age of a missing backup
		// cannot be established, so keep every Backup object for now.
		contextLogger.Info("Skipping deletion of backups not in the catalog: " +
			"cannot determine the oldest catalog backup")
		return nil
	}

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
			backupTime, err := parseBackupIDTime(backup.Status.BackupID)
			if err != nil {
				contextLogger.Info("Skipping deletion of backup not in the catalog: "+
					"cannot parse backup ID",
					"backup", backup.Name, "backupID", backup.Status.BackupID)
				continue
			}

			if !backupTime.Before(oldestCatalogBackup) {
				// The backup is newer than the oldest entry still in the
				// catalog, so it is still inside the retention window and
				// the catalog is expected to list it. Its absence means
				// the listing came back short; keep the object for the
				// next cycle instead of deleting a valid backup.
				contextLogger.Info("Skipping deletion of backup not in the catalog: "+
					"backup is newer than the oldest catalog entry",
					"backup", backup.Name, "backupID", backup.Status.BackupID)
				continue
			}

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

// backupIDTimeLayouts are the formats a Barman backup ID can have.
// Barman writes IDs as YYYYMMDDTHHMMSS, while some callers drop the "T".
var backupIDTimeLayouts = []string{"20060102T150405", "20060102150405"}

// parseBackupIDTime parses the timestamp carried by a Barman backup ID.
func parseBackupIDTime(backupID string) (time.Time, error) {
	for _, layout := range backupIDTimeLayouts {
		if t, err := time.Parse(layout, backupID); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized backup ID format: %q", backupID)
}

// oldestBackupTime returns the oldest backup time among the given catalog
// backup IDs. It returns an error when none of the IDs can be parsed.
func oldestBackupTime(backupIDs []string) (time.Time, error) {
	var oldest time.Time

	for _, backupID := range backupIDs {
		backupTime, err := parseBackupIDTime(backupID)
		if err != nil {
			continue
		}

		if oldest.IsZero() || backupTime.Before(oldest) {
			oldest = backupTime
		}
	}

	if oldest.IsZero() {
		return time.Time{}, fmt.Errorf("no parsable backup ID in the catalog")
	}

	return oldest, nil
}

// useSameBackupLocation checks whether the given backup was taken using the same configuration as provided
func useSameBackupLocation(backup *cnpgv1.BackupStatus, cluster *cnpgv1.Cluster) bool {
	if backup.Method != cnpgv1.BackupMethodPlugin {
		return false
	}

	meta := newBackupResultMetadataFromMap(backup.PluginMetadata)
	return meta.clusterUID == string(cluster.UID) && meta.pluginName == metadata.PluginName
}
