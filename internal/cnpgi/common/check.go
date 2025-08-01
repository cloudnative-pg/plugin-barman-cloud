package common

import (
	"context"

	"github.com/cloudnative-pg/barman-cloud/pkg/archiver"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/machinery/pkg/log"
)

// CheckBackupDestination checks if the backup destination is suitable
// to archive WALs
func CheckBackupDestination(
	ctx context.Context,
	barmanConfiguration *cnpgv1.BarmanObjectStoreConfiguration,
	barmanArchiver *archiver.WALArchiver,
	serverName string,
) error {
	contextLogger := log.FromContext(ctx)
	contextLogger.Info(
		"Checking backup destination with barman-cloud-wal-archive",
		"serverName", serverName)

	// Get WAL archive options
	checkWalOptions, err := barmanArchiver.BarmanCloudCheckWalArchiveOptions(
		ctx, barmanConfiguration, serverName)
	if err != nil {
		log.Error(err, "while getting barman-cloud-wal-archive options")
		return err
	}

	// Check if we're ok to archive in the desired destination
	return barmanArchiver.CheckWalArchiveDestination(ctx, checkWalOptions)
}
