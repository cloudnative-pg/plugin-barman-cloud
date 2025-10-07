/*
Copyright © contributors to CloudNativePG, established as
CloudNativePG a Series of LF Projects, LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

SPDX-License-Identifier: Apache-2.0
*/

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
