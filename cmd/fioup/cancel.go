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
		Use:   "cancel",
		Short: "Cancel the current update operation",
		Run: func(cmd *cobra.Command, args []string) {
			doCancel(cmd, &opts)
		},
		Args: cobra.NoArgs,
	}
	rootCmd.AddCommand(cmd)
}

func doCancel(cmd *cobra.Command, opts *update.UpdateOptions) {
	err := update.CancelPendingUpdate(cmd.Context(), config, opts)
	DieNotNil(err, "Failed to perform cancel")
	log.Info().Msgf("Cancel operation complete")
}
