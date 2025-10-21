// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"fmt"
	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/fioup/pkg/state"
	"strconv"
	"strings"

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
	}

	cmd.Flags().BoolVar(&opts.syncCurrent, "sync-current", false, "Sync the currently installed target if no version is specified.")
	rootCmd.AddCommand(cmd)
}

func doUpdate(cmd *cobra.Command, opts *updateOptions) {
	DieNotNil(api.Update(cmd.Context(), config, opts.version,
		api.WithForceUpdate(true),
		api.WithSyncCurrent(opts.syncCurrent),
		api.WithPreStateActionHandler(func(action state.ActionName, ctx *state.UpdateInfo) {
			fmt.Printf("[%d/%d] %s ... ", ctx.CurrentStateNum, ctx.TotalStates, action)
		}),
		api.WithPostStateActionHandler(func(action state.ActionName, ctx *state.UpdateInfo) {
			switch action {
			case "Checking":
				{
					switch ctx.Mode {
					case state.UpdateModeSync:
						fmt.Printf("sync the current target; version: %d, apps: %s\n", ctx.ToTarget.Version, strings.Join(ctx.ToTarget.AppNames(), ","))
					}
				}
			case "Initializing":
				{
					fmt.Printf("fetch: n apps, %s, %d blobs; remove: [tbd]; add: [tbd]\n",
						compose.FormatBytesInt64(ctx.Diff.ToFetch.Bytes),
						ctx.Diff.ToFetch.Blobs)
				}
			case "Fetching":
				{
					if ctx.Diff.ToFetch.Bytes == 0 {
						fmt.Printf("nothing to fetch\n")
					} else {
						fmt.Printf("fetched %s in total\n", compose.FormatBytesInt64(ctx.Diff.ToFetch.Bytes))
					}
				}
			}
		})))
}

func addCommonOptions(cmd *cobra.Command, opts *commonOptions) {
	cmd.Flags().BoolVar(&opts.enableTuf, "tuf", false, "Enable TUF metadata checking, instead of reading targets.json directly.")
	_ = cmd.Flags().MarkHidden("tuf")
}
