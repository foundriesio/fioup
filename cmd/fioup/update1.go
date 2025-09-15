// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"github.com/foundriesio/fioup/pkg/fioup"
	"github.com/spf13/cobra"
	"strconv"
)

type (
	updateOptions struct {
		version int
	}
)

func init() {
	opts := updateOptions{
		version: -1,
	}

	cmd := &cobra.Command{
		Use:   "update1 <target_name>",
		Short: "Update TUF metadata, download and install the selected target",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				var err error
				opts.version, err = strconv.Atoi(args[0])
				DieNotNil(err, "invalid version number")
			}
			doUpdate1(cmd, &opts)
		},
		Args: cobra.RangeArgs(0, 1),
	}
	rootCmd.AddCommand(cmd)
}

func doUpdate1(cmd *cobra.Command, opts *updateOptions) {
	DieNotNil(fioup.Update(cmd.Context(), config1, opts.version))
}
