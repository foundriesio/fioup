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
		Use:   "daemon",
		Short: "Start the update agent daemon",
		Run: func(cmd *cobra.Command, args []string) {
			doDaemon(cmd, &opts)
		},
		Args: cobra.NoArgs,
	}
	addCommonOptions(cmd, &opts)
	rootCmd.AddCommand(cmd)
}

func doDaemon(cmd *cobra.Command, opts *update.UpdateOptions) {
	update.Daemon(cmd.Context(), config, opts)
	log.Info().Msgf("Daemon exited")
}
