package healthcheck

import (
	"fmt"
	"os"
	"path"

	"github.com/cloudnative-pg/machinery/pkg/log"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/metadata"
)

// NewCmd returns the healthcheck command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "healthcheck",
		Short: "healthcheck commands",
	}

	cmd.AddCommand(unixHealthCheck())

	return cmd
}

func unixHealthCheck() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unix",
		Short: "unix healthcheck",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dialPath := fmt.Sprintf("unix://%s", path.Join("/plugins", metadata.PluginName))
			cli, cliErr := grpc.NewClient(dialPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if cliErr != nil {
				log.Error(cliErr, "while building the client")
				return cliErr
			}

			healthCli := grpc_health_v1.NewHealthClient(cli)
			res, healthErr := healthCli.Check(
				cmd.Context(),
				&grpc_health_v1.HealthCheckRequest{},
			)
			if healthErr != nil {
				log.Error(healthErr, "while executing the healthcheck call")
				return healthErr
			}

			if res.Status == grpc_health_v1.HealthCheckResponse_SERVING {
				log.Trace("healthcheck response OK")
				os.Exit(0)
				return nil
			}

			log.Error(fmt.Errorf("unexpected healthcheck response: %v", res.Status),
				"while processing healthcheck response")

			// exit code 1 is returned when we exit from the function with an error
			switch res.Status {
			case grpc_health_v1.HealthCheckResponse_UNKNOWN:
				os.Exit(2)
			case grpc_health_v1.HealthCheckResponse_NOT_SERVING:
				os.Exit(3)
			default:
				os.Exit(125)
			}

			return nil
		},
	}

	return cmd
}
