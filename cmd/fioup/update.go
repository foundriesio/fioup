// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"github.com/foundriesio/fioup/internal/update"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func addCommonOptions(cmd *cobra.Command, opts *update.UpdateOptions) {
	cmd.Flags().BoolVar(&opts.EnableTuf, "tuf", false, "Enable TUF metadata checking, instead of reading targets.json directly.")
	_ = cmd.Flags().MarkHidden("tuf")
}

func init() {
	opts := update.UpdateOptions{}
	cmd := &cobra.Command{
		Use:   "update <target_name_or_version>",
		Short: "Update TUF metadata, download and install the selected target",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				opts.TargetId = args[0]
			}
			doUpdate(cmd, &opts)
		},
		Args: cobra.RangeArgs(0, 1),
	}
	addCommonOptions(cmd, &opts)
	rootCmd.AddCommand(cmd)
}

func doUpdate(cmd *cobra.Command, opts *update.UpdateOptions) {
	opts.DoCheck = true
	opts.DoFetch = true
	opts.DoInstall = true
	opts.DoStart = true
	err := update.Update(cmd.Context(), config, opts)
	DieNotNil(err, "Failed to perform update")
	log.Info().Msgf("Update operation complete")
}
