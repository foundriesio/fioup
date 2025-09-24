// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"log/slog"

	"github.com/foundriesio/fioup/internal/update"
	"github.com/spf13/cobra"
)

func init() {
	opts := update.UpdateOptions{}
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the update. A fetch operation must be performed first.",
		Run: func(cmd *cobra.Command, args []string) {
			doInstall(cmd, &opts)
		},
		Args: cobra.NoArgs,
	}
	rootCmd.AddCommand(cmd)
}

func doInstall(cmd *cobra.Command, opts *update.UpdateOptions) {
	opts.DoInstall = true
	err := update.Update(cmd.Context(), config, opts)
	DieNotNil(err, "Failed to perform install operation")
	slog.Info("Install operation complete")
}
