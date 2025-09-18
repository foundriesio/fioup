// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"github.com/foundriesio/fioup/pkg/fioup"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "start1 [<target>]",
		Short: "Start previously fetched and installed update or resume interrupted start",
		Run: func(cmd *cobra.Command, args []string) {
			doStart1(cmd)
		},
		Args: cobra.NoArgs,
	}
	rootCmd.AddCommand(cmd)
}

func doStart1(cmd *cobra.Command) {
	DieNotNil(fioup.Start(cmd.Context(), config1))
}
