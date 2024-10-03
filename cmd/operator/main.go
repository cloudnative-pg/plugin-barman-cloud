// Package main is the entrypoint of operator plugin
package main

import (
	"fmt"
	"os"

	"github.com/cloudnative-pg/machinery/pkg/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator"
)

func main() {
	cobra.EnableTraverseRunHooks = true

	logFlags := &log.Flags{}
	rootCmd := &cobra.Command{
		Use:   "plugin-barman-cloud",
		Short: "Starts the BarmanObjectStore reconciler and the Barman Cloud CNPG-i plugin",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(viper.GetString("sidecar-image")) == 0 {
				return fmt.Errorf("missing required SIDECAR_IMAGE environment variable")
			}

			return operator.Start(cmd.Context())
		},
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			logFlags.ConfigureLogging()
			return nil
		},
	}

	logFlags.AddFlags(rootCmd.PersistentFlags())

	rootCmd.Flags().String("metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	_ = viper.BindPFlag("metrics-bind-address", rootCmd.Flags().Lookup("metrics-bind-address"))

	rootCmd.Flags().String("health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	_ = viper.BindPFlag("health-probe-bind-address", rootCmd.Flags().Lookup("health-probe-bind-address"))

	rootCmd.Flags().Bool("leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	_ = viper.BindPFlag("leader-elect", rootCmd.Flags().Lookup("leader-elect"))

	rootCmd.Flags().Bool("metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	_ = viper.BindPFlag("metrics-secure", rootCmd.Flags().Lookup("metrics-secure"))

	rootCmd.Flags().Bool("enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	_ = viper.BindPFlag("enable-http2", rootCmd.Flags().Lookup("enable-http2"))

	rootCmd.Flags().String(
		"plugin-path",
		"",
		"The plugins socket path",
	)
	_ = viper.BindPFlag("plugin-path", rootCmd.Flags().Lookup("plugin-path"))

	rootCmd.Flags().String(
		"server-cert",
		"",
		"The public key to be used for the server process",
	)
	_ = viper.BindPFlag("server-cert", rootCmd.Flags().Lookup("server-cert"))

	rootCmd.Flags().String(
		"server-key",
		"",
		"The key to be used for the server process",
	)
	_ = viper.BindPFlag("server-key", rootCmd.Flags().Lookup("server-key"))

	rootCmd.Flags().String(
		"client-cert",
		"",
		"The client public key to verify the connection",
	)
	_ = viper.BindPFlag("client-cert", rootCmd.Flags().Lookup("client-cert"))

	rootCmd.Flags().String(
		"server-address",
		"",
		"The address where to listen (i.e. 0:9090)",
	)
	_ = viper.BindPFlag("server-address", rootCmd.Flags().Lookup("server-address"))

	_ = viper.BindEnv("sidecar-image", "SIDECAR_IMAGE")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
