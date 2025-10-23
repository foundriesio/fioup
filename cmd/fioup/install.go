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
		Use:   "install",
		Short: "Install previously fetched update or resume interrupted install",
		Run: func(cmd *cobra.Command, args []string) {
			doInstall(cmd)
		},
		Args: cobra.NoArgs,
	}
	rootCmd.AddCommand(cmd)
}

func doInstall(cmd *cobra.Command) {
	DieNotNil(api.Install(cmd.Context(), config,
		append(updateHandlers,
			api.WithInstallProgressHandler(update.GetInstallProgressPrinter(update.WithIndentation(8))),
		)...,
	))
}
