// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"fmt"
	"strconv"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/fioup/pkg/api"
	"github.com/spf13/cobra"
)

type (
	diffOptions struct {
		commonOptions
		toVersion   int
		fromVersion int
		blobs       bool
	}
)

func init() {
	opts := &diffOptions{}
	cmd := &cobra.Command{
		Use:   "diff <to-version> [<from-version>, use current if not specified>]",
		Short: "Show differences between two versions",
		Run: func(cmd *cobra.Command, args []string) {
			var err error
			opts.toVersion, err = strconv.Atoi(args[0])
			DieNotNil(err, "invalid to-version number specified")
			if len(args) > 1 {
				opts.fromVersion, err = strconv.Atoi(args[1])
				DieNotNil(err, "invalid from-version number specified")
			} else {
				opts.fromVersion = -1 // current version is implied if not specified
			}
			doDiff(cmd, opts)
		},
		Args: cobra.RangeArgs(1, 2),
	}
	addCommonOptions(cmd, &opts.commonOptions)
	cmd.Flags().BoolVarP(&opts.blobs, "blobs", "b", false, "Show diff blob details")
	rootCmd.AddCommand(cmd)
}

func doDiff(cmd *cobra.Command, opts *diffOptions) {
	diff, err := api.Diff(cmd.Context(), config, opts.fromVersion, opts.toVersion, api.WithTUFEnabled(opts.enableTuf))
	DieNotNil(err, "failed to obtain diff:")
	fmt.Printf("On wire size: %s\n", compose.FormatBytesInt64(diff.WireSize))
	fmt.Printf("On disk size: %s\n", compose.FormatBytesInt64(diff.DiskSize))
	fmt.Printf("Blob count:   %d\n", len(diff.Blobs))
	if opts.blobs {
		fmt.Println("Blobs:")
		for baseURL, b := range diff.Blobs {
			fmt.Println("- " + baseURL)
			for _, b := range b {
				fmt.Printf("\t- %s: on wire %9s, on disk %s\n", b.Descriptor.Digest,
					compose.FormatBytesInt64(b.Descriptor.Size),
					compose.FormatBytesInt64(b.StoreSize+b.RuntimeSize))
			}
		}
	}
}
