// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/pkg/status"
	"github.com/spf13/cobra"
)

type (
	statusReport struct {
		// Represents the current status of apps from the last successful update.
		// If no successful update exists, falls back to the status of apps detected locally.
		CurrentStatus *status.CurrentStatus `json:"current_status"`
		// Status of a pending or last update operation, if any
		UpdateStatus *status.UpdateStatus `json:"update_status,omitempty"`
	}
	statusOptions struct {
		Format string
	}
)

func init() {
	opts := statusOptions{}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show information about the currently running target/apps and pending or last update status, if any",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&opts.Format, "format", "text", "Format the output. Values: [text | json]")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		switch opts.Format {
		case "text", "json":
			doStatus(cmd, &opts)
		default:
			return fmt.Errorf("invalid value for --format: %s (must be text or json)", opts.Format)
		}
		return nil
	}

	rootCmd.AddCommand(cmd)
}

func doStatus(cmd *cobra.Command, opts *statusOptions) {
	cs, err := status.GetCurrentStatus(cmd.Context(), config.ComposeConfig())
	DieNotNil(err, "failed to get current status")
	us, err := status.GetUpdateStatus(config.ComposeConfig())
	DieNotNil(err, "failed to get update status")

	if opts.Format == "json" {
		if b, err := json.Marshal(statusReport{CurrentStatus: cs, UpdateStatus: us}); err != nil {
			DieNotNil(err, "failed to marshal status report")
		} else {
			fmt.Println(string(b))
		}
		return
	}
	fmt.Println("Current status:")
	fmt.Printf("  Target ID:\t%s\n", cs.TargetID)
	fmt.Printf("  Update ID:\t%s\n", cs.UpdateID)
	fmt.Printf("  Completed at:\t%s\n", cs.CompletedAt.Local().Format(time.DateTime))
	fmt.Println("  Apps:")
	for _, app := range cs.AppStatuses {
		fmt.Printf("\t\t[%s]: \n", app.Name)
		fmt.Printf("\t\t  %s\n", app.URI)
		fmt.Printf("\t\t  fetched:%v; installed:%v; running:%v\n", app.Fetched, app.Installed, app.Running)
		fmt.Println()
	}
	ongoing := true
	if us.State == update.StateCompleted || us.State == update.StateCanceled || us.State == update.StateFailed {
		fmt.Println("Last update status:")
		ongoing = false
	} else {
		fmt.Println("Ongoing update status:")
	}
	fmt.Printf("  Target ID:\t%s\n", us.TargetID)
	fmt.Printf("  Update ID:\t%s\n", us.ID)
	fmt.Printf("  State:\t%s\n", us.State)
	fmt.Printf("  Started at:\t%s\n", us.StartTime.Local().Format(time.DateTime))
	if ongoing {
		fmt.Printf("  Updated at:\t%s\n", us.StartTime.Local().Format(time.DateTime))
	} else {
		fmt.Printf("  Completed at:\t%s\n", us.StartTime.Local().Format(time.DateTime))
	}

	fmt.Println("  Apps:")
	for _, app := range us.Apps {
		fmt.Printf("\t\t- %s\n", app)
	}
	fmt.Printf("  Update size:\t%s, %d blobs\n", compose.FormatBytesInt64(us.Size.Bytes), us.Size.NumBlobs)
	if us.FetchedSize.Bytes < us.Size.Bytes {
		fmt.Printf("  Fetched:\t%s, %d blobs\n", compose.FormatBytesInt64(us.FetchedSize.Bytes), us.FetchedSize.NumBlobs)
	}
	if ongoing {
		fmt.Printf("  Progress:\t%d\n", us.Progress)
	}
}
