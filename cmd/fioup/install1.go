// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"github.com/foundriesio/fioup/pkg/fioup"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "install1 [<target>]",
		Short: "Install previously fetched update or resume interrupted install",
		Run: func(cmd *cobra.Command, args []string) {
			doInstall1(cmd)
		},
		Args: cobra.NoArgs,
	}
	rootCmd.AddCommand(cmd)
}

func doInstall1(cmd *cobra.Command) {
	DieNotNil(fioup.Install(cmd.Context(), config1))
}
