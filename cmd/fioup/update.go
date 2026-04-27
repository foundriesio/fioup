// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"fmt"
	"strconv"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/pkg/api"
	"github.com/spf13/cobra"
)

type (
	commonOptions struct {
		enableTuf bool
	}
	updateOptions struct {
		version     int
		syncCurrent bool
	}
)

func init() {
	opts := updateOptions{
		version:     -1,
		syncCurrent: false,
	}

	cmd := &cobra.Command{
		Use:   "update [<version>]",
		Short: "Update target metadata, download, install, and start the specified target or the latest one if not specified",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				var err error
				opts.version, err = strconv.Atoi(args[0])
				DieNotNil(err, "invalid version number specified")
				if opts.syncCurrent {
					DieNotNil(fmt.Errorf("--sync-current cannot be used when a version is specified"))
				}
			}
			doUpdate(cmd, &opts)
		},
		Args: cobra.RangeArgs(0, 1),
		Annotations: map[string]string{
			lockFlagKey: "true",
		},
	}

	cmd.Flags().BoolVar(&opts.syncCurrent, "sync-current", false, "Sync the currently installed target if no version is specified.")
	rootCmd.AddCommand(cmd)
}

func doUpdate(cmd *cobra.Command, opts *updateOptions) {
	DieNotNil(api.Update(cmd.Context(), config, opts.version,
		append(updateHandlers,
			api.WithForceUpdate(true),
			api.WithSyncCurrent(opts.syncCurrent),
			api.WithFetchProgressHandler(update.GetFetchProgressPrinter(update.WithIndentation(8))),
			api.WithInstallProgressHandler(update.GetInstallProgressPrinter(update.WithIndentation(8))),
			api.WithStartProgressHandler(appStartHandler),
		)...))
}

func addCommonOptions(cmd *cobra.Command, opts *commonOptions) {
	cmd.Flags().BoolVar(&opts.enableTuf, "tuf", false, "Enable TUF metadata checking, instead of reading targets.json directly.")
	_ = cmd.Flags().MarkHidden("tuf")
}
