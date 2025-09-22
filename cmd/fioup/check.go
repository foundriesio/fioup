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
		Use:   "check",
		Short: "Update TUF metadata",
		Run: func(cmd *cobra.Command, args []string) {
			doCheck(cmd, &opts)
		},
		Args: cobra.NoArgs,
	}
	addCommonOptions(cmd, &opts)
	rootCmd.AddCommand(cmd)
}

func doCheck(cmd *cobra.Command, opts *update.UpdateOptions) {
	opts.DoCheck = true
	err := update.Update(cmd.Context(), config, opts)
	DieNotNil(err, "Failed to perform check operation")
	log.Info().Msgf("Check operation complete")
}
