// Package operator is the entrypoint of operator plugin
package operator

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator"
)

// NewCmd creates a new operator command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operator",
		Short: "Starts the BarmanObjectStore reconciler and the Barman Cloud CNPG-i plugin",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(viper.GetString("sidecar-image")) == 0 {
				return fmt.Errorf("missing required SIDECAR_IMAGE environment variable")
			}

			return operator.Start(cmd.Context())
		},
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	}

	cmd.Flags().String("metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	_ = viper.BindPFlag("metrics-bind-address", cmd.Flags().Lookup("metrics-bind-address"))

	cmd.Flags().String("health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	_ = viper.BindPFlag("health-probe-bind-address", cmd.Flags().Lookup("health-probe-bind-address"))

	cmd.Flags().Bool("leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	_ = viper.BindPFlag("leader-elect", cmd.Flags().Lookup("leader-elect"))

	cmd.Flags().Bool("metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	_ = viper.BindPFlag("metrics-secure", cmd.Flags().Lookup("metrics-secure"))

	cmd.Flags().Bool("enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	_ = viper.BindPFlag("enable-http2", cmd.Flags().Lookup("enable-http2"))

	cmd.Flags().String(
		"plugin-path",
		"",
		"The plugins socket path",
	)
	_ = viper.BindPFlag("plugin-path", cmd.Flags().Lookup("plugin-path"))

	cmd.Flags().String(
		"server-cert",
		"",
		"The public key to be used for the server process",
	)
	_ = viper.BindPFlag("server-cert", cmd.Flags().Lookup("server-cert"))

	cmd.Flags().String(
		"server-key",
		"",
		"The key to be used for the server process",
	)
	_ = viper.BindPFlag("server-key", cmd.Flags().Lookup("server-key"))

	cmd.Flags().String(
		"client-cert",
		"",
		"The client public key to verify the connection",
	)
	_ = viper.BindPFlag("client-cert", cmd.Flags().Lookup("client-cert"))

	cmd.Flags().String(
		"server-address",
		"",
		"The address where to listen (i.e. 0:9090)",
	)
	_ = viper.BindPFlag("server-address", cmd.Flags().Lookup("server-address"))

	_ = viper.BindEnv("sidecar-image", "SIDECAR_IMAGE")

	_ = viper.BindEnv("pprof-server", "PPROF_SERVER")

	return cmd
}
