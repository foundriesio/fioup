// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"github.com/foundriesio/fioup/internal/update"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	opts := update.UpdateOptions{}
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start execution of the updated target. A install operation must be performed first.",
		Run: func(cmd *cobra.Command, args []string) {
			doStart(cmd, &opts)
		},
		Args: cobra.NoArgs,
	}
	rootCmd.AddCommand(cmd)
}

func doStart(cmd *cobra.Command, opts *update.UpdateOptions) {
	opts.DoStart = true
	err := update.Update(cmd.Context(), config, opts)
	DieNotNil(err, "Failed to perform start operation")
	log.Info().Msgf("Start operation complete")
}
