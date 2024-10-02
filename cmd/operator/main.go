// Package main is the entrypoint of operator plugin
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudnative-pg/machinery/pkg/log"
	"github.com/sourcegraph/conc/pool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/operator/manager"
)

func main() {
	cobra.EnableTraverseRunHooks = true

	logFlags := &log.Flags{}
	rootCmd := &cobra.Command{
		Use: "plugin-barman-cloud",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			logFlags.ConfigureLogging()
			return nil
		},
	}

	logFlags.AddFlags(rootCmd.PersistentFlags())
	rootCmd.AddCommand(newOperatorCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func newOperatorCommand() *cobra.Command {
	cmd := operator.NewCommand()
	cmd.Use = "operator"
	cmd.Short = "Starts the BarmanObjectStore reconciler and the Barman Cloud CNPG-i plugin"
	grpcServer := cmd.RunE

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		operatorPool := pool.
			New().
			WithContext(cmd.Context()).
			WithCancelOnError().
			WithFirstError()
		operatorPool.Go(func(ctx context.Context) error {
			cmd.SetContext(ctx)

			if len(viper.GetString("sidecar-image")) == 0 {
				return fmt.Errorf("missing required SIDECAR_IMAGE environment variable")
			}

			err := grpcServer(cmd, args)
			return err
		})
		operatorPool.Go(manager.Start)
		return operatorPool.Wait()
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

	_ = viper.BindEnv("sidecar-image", "SIDECAR_IMAGE")

	return cmd
}
