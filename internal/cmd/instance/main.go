// Package instance is the entrypoint of instance plugin
package instance

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/instance"
)

// NewCmd creates a new instance command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instance",
		Short: "Starts the Barman Cloud CNPG-I sidecar plugin",
		RunE: func(cmd *cobra.Command, _ []string) error {
			requiredSettings := []string{
				"namespace",
				"barman-object-name",
				"cluster-name",
				"pod-name",
				"spool-directory",
				"server-name",
			}

			for _, k := range requiredSettings {
				if len(viper.GetString(k)) == 0 {
					return fmt.Errorf("missing required %s setting", k)
				}
			}

			return instance.Start(cmd.Context())
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
