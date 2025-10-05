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

	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/http"
	"github.com/cloudnative-pg/cnpg-i/pkg/backup"
	"github.com/cloudnative-pg/cnpg-i/pkg/metrics"
	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
	"google.golang.org/grpc"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/common"
)

// CNPGI is the implementation of the PostgreSQL sidecar
type CNPGI struct {
	Client         client.Client
	PGDataPath     string
	PGWALPath      string
	SpoolDirectory string
	// mutually exclusive with serverAddress
	PluginPath   string
	InstanceName string
}

// Start starts the GRPC service
func (c *CNPGI) Start(ctx context.Context) error {
	enrich := func(server *grpc.Server) error {
		wal.RegisterWALServer(server, common.WALServiceImplementation{
			InstanceName:   c.InstanceName,
			Client:         c.Client,
			SpoolDirectory: c.SpoolDirectory,
			PGDataPath:     c.PGDataPath,
			PGWALPath:      c.PGWALPath,
		})
		backup.RegisterBackupServer(server, BackupServiceImplementation{
			Client:       c.Client,
			InstanceName: c.InstanceName,
		})
		metrics.RegisterMetricsServer(server, &metricsImpl{
			Client: c.Client,
		})
		common.AddHealthCheck(server)
		return nil
	}

	srv := http.Server{
		IdentityImpl: IdentityImplementation{
			Client: c.Client,
		},
		Enrichers:  []http.ServerEnricher{enrich},
		PluginPath: c.PluginPath,
	}

	return srv.Start(ctx)
}
