// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"fmt"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/pkg/api"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {

	cmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel the current update operation",
		Run: func(cmd *cobra.Command, args []string) {
			doCancel(cmd)
		},
		Args: cobra.NoArgs,
	}
	rootCmd.AddCommand(cmd)
}

func doCancel(cmd *cobra.Command) {
	targetID, err := api.Cancel(cmd.Context(), config)
	if errors.Is(err, update.ErrUpdateNotFound) {
		fmt.Println("No update in progress to cancel")
	} else {
		DieNotNil(err, "failed to cancel update")
		fmt.Println("Cancelled update to target ", targetID)
	}
}
