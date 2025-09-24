// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Commit string

func init() {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display version of this tool",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Commit)
		},
		Args: cobra.NoArgs,
	}
	rootCmd.AddCommand(cmd)
}
