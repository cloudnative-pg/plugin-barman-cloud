package main

import (
	"fmt"
	"os"


	"github.com/cloudnative-pg/machinery/pkg/log"
	"github.com/spf13/cobra"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cmd/instance"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cmd/operator"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cmd/restore"
)

func main() {
	cobra.EnableTraverseRunHooks = true

	logFlags := &log.Flags{}
	rootCmd := &cobra.Command{
		Use: "manager [cmd]",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			logFlags.ConfigureLogging()
			return nil
		},
	}

	logFlags.AddFlags(rootCmd.PersistentFlags())

	rootCmd.AddCommand(instance.NewCmd())
	rootCmd.AddCommand(operator.NewCmd())
	rootCmd.AddCommand(restore.NewCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
