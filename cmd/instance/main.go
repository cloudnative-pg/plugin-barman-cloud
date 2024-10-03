// Package main is the entrypoint of operator plugin
package main

import (
	"fmt"
	"os"

	"github.com/cloudnative-pg/machinery/pkg/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/instance"
)

func main() {
	cobra.EnableTraverseRunHooks = true

	logFlags := &log.Flags{}
	rootCmd := &cobra.Command{
		Use:   "instance",
		Short: "Starts the Barman Cloud CNPG-i sidecar plugin",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			logFlags.ConfigureLogging()
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			requiredSettings := []string{
				"namespace",
				"barman-object-name",
				"cluster-name",
				"pod-name",
				"spool-directory",
			}

			for _, k := range requiredSettings {
				if len(viper.GetString(k)) == 0 {
					return fmt.Errorf("missing required %s setting", k)
				}
			}

			return instance.Start(cmd.Context())
		},
	}

	logFlags.AddFlags(rootCmd.PersistentFlags())

	_ = viper.BindEnv("namespace", "NAMESPACE")
	_ = viper.BindEnv("barman-object-name", "BARMAN_OBJECT_NAME")
	_ = viper.BindEnv("cluster-name", "CLUSTER_NAME")
	_ = viper.BindEnv("pod-name", "POD_NAME")
	_ = viper.BindEnv("pgdata", "PGDATA")
	_ = viper.BindEnv("spool-directory", "SPOOL_DIRECTORY")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
