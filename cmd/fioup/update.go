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
	commonOptions struct {
		enableTuf bool
	}
	updateOptions struct {
		version int
	}
)

func init() {
	opts := updateOptions{
		version: -1,
	}

	cmd := &cobra.Command{
		Use:   "update [<version>]",
		Short: "Update target metadata, download, install, and start the specified target or the latest one if not specified",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				var err error
				opts.version, err = strconv.Atoi(args[0])
				if err != nil {
					cobra.CheckErr(fmt.Errorf("invalid version number: %w", err))
				}
			}
			doUpdate(cmd, &opts)
		},
		Args: cobra.RangeArgs(0, 1),
	}
	rootCmd.AddCommand(cmd)
}

func doUpdate(cmd *cobra.Command, opts *updateOptions) {
	DieNotNil(api.Update(cmd.Context(), config, opts.version, false))
}

func addCommonOptions(cmd *cobra.Command, opts *commonOptions) {
	cmd.Flags().BoolVar(&opts.enableTuf, "tuf", false, "Enable TUF metadata checking, instead of reading targets.json directly.")
	_ = cmd.Flags().MarkHidden("tuf")
}
