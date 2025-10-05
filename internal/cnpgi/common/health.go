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

	"github.com/cloudnative-pg/machinery/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// AddHealthCheck adds a health check service to the gRPC server with the tag 'plugin-barman-cloud'
func AddHealthCheck(server *grpc.Server) {
	grpc_health_v1.RegisterHealthServer(server, &healthServer{}) // replaces default registration
}

type healthServer struct {
	grpc_health_v1.UnimplementedHealthServer
}

// Check is the response handle for the healthcheck request
func (h healthServer) Check(
	ctx context.Context,
	_ *grpc_health_v1.HealthCheckRequest,
) (*grpc_health_v1.HealthCheckResponse, error) {
	contextLogger := log.FromContext(ctx)
	contextLogger.Trace("serving health check response")
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}
