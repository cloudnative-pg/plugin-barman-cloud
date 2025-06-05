package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/cloudnative-pg/machinery/pkg/log"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cmd/healthcheck"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cmd/instance"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cmd/operator"
	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cmd/restore"
)

func main() {
	cobra.EnableTraverseRunHooks = true

	logFlags := &log.Flags{}
	rootCmd := &cobra.Command{
		Use: "manager [cmd]",
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			logFlags.ConfigureLogging()
			cmd.SetContext(log.IntoContext(cmd.Context(), log.GetLogger()))
		},
	}

	logFlags.AddFlags(rootCmd.PersistentFlags())

	rootCmd.AddCommand(instance.NewCmd())
	rootCmd.AddCommand(operator.NewCmd())
	rootCmd.AddCommand(restore.NewCmd())
	rootCmd.AddCommand(healthcheck.NewCmd())

	if err := rootCmd.ExecuteContext(ctrl.SetupSignalHandler()); err != nil {
		if !errors.Is(err, context.Canceled) {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
