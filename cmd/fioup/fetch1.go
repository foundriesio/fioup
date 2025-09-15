// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"github.com/foundriesio/fioup/pkg/fioup"
	"github.com/spf13/cobra"
	"strconv"
)

type (
	fetchOptions struct {
		version int
	}
)

func init() {
	opts := fetchOptions{
		version: -1,
	}

	cmd := &cobra.Command{
		Use:   "fetch1 [<target>]",
		Short: "Stage and fetch update or resume interrupted fetch",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				var err error
				opts.version, err = strconv.Atoi(args[0])
				DieNotNil(err, "invalid version number")
			}
			doFetch1(cmd, &opts)
		},
		Args: cobra.RangeArgs(0, 1),
	}
	rootCmd.AddCommand(cmd)
}

func doFetch1(cmd *cobra.Command, opts *fetchOptions) {
	DieNotNil(fioup.Fetch(cmd.Context(), config1, opts.version))
}
