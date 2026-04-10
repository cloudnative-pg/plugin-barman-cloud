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

package common

import (
	"context"

	"github.com/cloudnative-pg/barman-cloud/pkg/archiver"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/machinery/pkg/log"
)

// CheckBackupDestination checks if the backup destination is suitable
// to archive WALs.
//
// The timeline parameter, when > 0, is passed to
// barman-cloud-check-wal-archive via --timeline so that WAL from earlier
// timelines (expected after a failover) does not cause the check to fail.
func CheckBackupDestination(
	ctx context.Context,
	barmanConfiguration *cnpgv1.BarmanObjectStoreConfiguration,
	barmanArchiver *archiver.WALArchiver,
	serverName string,
	timeline int,
) error {
	contextLogger := log.FromContext(ctx)
	contextLogger.Info(
		"Checking backup destination with barman-cloud-check-wal-archive",
		"serverName", serverName,
		"timeline", timeline)

	// Build options, passing --timeline when available so that WAL from
	// earlier timelines in the archive is accepted after a failover.
	var opts []archiver.CheckWalArchiveOption
	if timeline > 0 {
		opts = append(opts, archiver.WithTimeline(timeline))
	}

	checkWalOptions, err := barmanArchiver.BarmanCloudCheckWalArchiveOptions(
		ctx, barmanConfiguration, serverName, opts...)
	if err != nil {
		log.Error(err, "while getting barman-cloud-wal-archive options")
		return err
	}

	// Check if we're ok to archive in the desired destination
	return barmanArchiver.CheckWalArchiveDestination(ctx, checkWalOptions)
}
