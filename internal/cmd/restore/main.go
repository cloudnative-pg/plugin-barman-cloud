// Package restore is the entrypoint of restore capabilities
package restore

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/restore"
)

// NewCmd creates the "restore" subcommand
func NewCmd() *cobra.Command {
	cobra.EnableTraverseRunHooks = true

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Starts the Barman Cloud CNPG-I sidecar plugin",
		RunE: func(cmd *cobra.Command, _ []string) error {
			requiredSettings := []string{
				"namespace",
				"cluster-name",
				"pod-name",
				"spool-directory",

				// IMPORTANT: barman-object-name and server-name are not required
				// to restore a cluster.
				"recovery-barman-object-name",
				"recovery-server-name",
			}

			for _, k := range requiredSettings {
				if len(viper.GetString(k)) == 0 {
					return fmt.Errorf("missing required %s setting", k)
				}
			}

			return restore.Start(cmd.Context())
		},
	}

	_ = viper.BindEnv("namespace", "NAMESPACE")
	_ = viper.BindEnv("cluster-name", "CLUSTER_NAME")
	_ = viper.BindEnv("pod-name", "POD_NAME")
	_ = viper.BindEnv("pgdata", "PGDATA")
	_ = viper.BindEnv("spool-directory", "SPOOL_DIRECTORY")

	_ = viper.BindEnv("barman-object-name", "BARMAN_OBJECT_NAME")
	_ = viper.BindEnv("server-name", "SERVER_NAME")

	_ = viper.BindEnv("recovery-barman-object-name", "RECOVERY_BARMAN_OBJECT_NAME")
	_ = viper.BindEnv("recovery-server-name", "RECOVERY_SERVER_NAME")

	return cmd
}
