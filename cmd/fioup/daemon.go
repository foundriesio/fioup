// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause
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
			doDaemon(&opts)
		},
		Args: cobra.NoArgs,
	}
	addCommonOptions(cmd, &opts)
	rootCmd.AddCommand(cmd)
}

func doDaemon(opts *update.UpdateOptions) {
	update.Daemon(config, opts)
	log.Info().Msgf("Daemon exited")
}
