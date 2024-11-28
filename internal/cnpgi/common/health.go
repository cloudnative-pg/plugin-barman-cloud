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
