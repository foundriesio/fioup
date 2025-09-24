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
		Use:   "fetch <target_name_or_version>",
		Short: "Fetch the update from the OTA server",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				opts.TargetId = args[0]
			}
			doFetch(cmd, &opts)
		},
		Args: cobra.RangeArgs(0, 1),
	}
	addCommonOptions(cmd, &opts)
	rootCmd.AddCommand(cmd)
}

func doFetch(cmd *cobra.Command, opts *update.UpdateOptions) {
	opts.DoFetch = true
	err := update.Update(cmd.Context(), config, opts)
	DieNotNil(err, "Failed to perform fetch operation")
	slog.Info("Fetch operation complete")
}
