// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"fmt"
	"strconv"

	"github.com/foundriesio/fioup/pkg/api"
	"github.com/spf13/cobra"
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
		Use:   "fetch [<version>]",
		Short: "Initialize and fetch update or resume interrupted one",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				var err error
				opts.version, err = strconv.Atoi(args[0])
				if err != nil {
					cobra.CheckErr(fmt.Errorf("invalid version number: %w", err))
				}
			}
			doFetch(cmd, &opts)
		},
		Args: cobra.RangeArgs(0, 1),
	}
	rootCmd.AddCommand(cmd)
}

func doFetch(cmd *cobra.Command, opts *fetchOptions) {
	DieNotNil(api.Fetch(cmd.Context(), config, opts.version))
}
