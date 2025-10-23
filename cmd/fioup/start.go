// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/pkg/api"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start previously fetched and installed update or resume interrupted start",
		Run: func(cmd *cobra.Command, args []string) {
			doStart(cmd)
		},
		Args: cobra.NoArgs,
	}
	rootCmd.AddCommand(cmd)
}

func doStart(cmd *cobra.Command) {
	DieNotNil(api.Start(cmd.Context(), config,
		append(updateHandlers,
			api.WithInstallProgressHandler(update.GetInstallProgressPrinter(update.WithIndentation(8))),
			api.WithStartProgressHandler(appStartHandler),
		)...,
	))
}
