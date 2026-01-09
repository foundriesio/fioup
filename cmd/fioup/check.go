// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"encoding/json"
	"fmt"

	"github.com/foundriesio/fioup/pkg/api"
	"github.com/foundriesio/fioup/pkg/status"
	"github.com/foundriesio/fioup/pkg/target"
	"github.com/spf13/cobra"
)

type (
	checkOptions struct {
		Format string
	}
)

func init() {
	cOpts := commonOptions{}
	opts := checkOptions{
		Format: "text",
	}
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Update TUF metadata",
		Args:  cobra.NoArgs,
		Annotations: map[string]string{
			lockFlagKey: "true",
		},
	}
	cmd.Flags().StringVar(&opts.Format, "format", "text", "Format the output. Values: [text | json]")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		switch opts.Format {
		case "text", "json":
			doCheck(cmd, &cOpts, &opts)
		default:
			return fmt.Errorf("invalid value for --format: %s (must be text or json)", opts.Format)
		}
		return nil
	}
	addCommonOptions(cmd, &cOpts)
	rootCmd.AddCommand(cmd)
}

func doCheck(cmd *cobra.Command, cOpts *commonOptions, opts *checkOptions) {
	targets, currentStatus, err := api.Check(cmd.Context(), config, api.WithTUF(cOpts.enableTuf))
	DieNotNil(err, "failed to check for updates")
	if opts.Format == "json" {
		printJsonResult(targets, currentStatus)
	} else {
		printTextResult(targets, currentStatus)
	}
}

func printJsonResult(targets target.Targets, currentStatus *status.CurrentStatus) {
	var areAppsInSync = true
	for _, app := range currentStatus.AppStatuses {
		if !app.Fetched || !app.Installed || !app.Running {
			areAppsInSync = false
		}
	}

	details := ""
	reason := ""
	selectedTarget := ""
	updateRequired := false
	if currentStatus.TargetID != targets.GetLatestTarget().ID {
		details = fmt.Sprintf("New version available: %s", targets.GetLatestTarget().ID)
		updateRequired = true
		selectedTarget = targets.GetLatestTarget().ID
		reason = "new version available"
	} else if !areAppsInSync {
		details = "You are running the latest version, but not all apps are in sync."
		updateRequired = true
		selectedTarget = targets.GetLatestTarget().ID
		reason = "apps out of sync"
	} else {
		details = "You are running the latest version."
	}

	result := struct {
		Targets       target.Targets        `json:"targets"`
		CurrentStatus *status.CurrentStatus `json:"current_status"`
		CheckResult   any                   `json:"check_result"`
	}{
		Targets:       targets,
		CurrentStatus: currentStatus,
		CheckResult: struct {
			AppsAreInSync  bool   `json:"apps_are_in_sync"`
			UpdateRequired bool   `json:"update_required"`
			Reason         string `json:"reason,omitempty"`
			SelectedTarget string `json:"selected_target,omitempty"`
			Details        string `json:"details"`
		}{
			AppsAreInSync:  areAppsInSync,
			UpdateRequired: updateRequired,
			Reason:         reason,
			SelectedTarget: selectedTarget,
			Details:        details,
		},
	}

	if b, err := json.Marshal(result); err != nil {
		DieNotNil(err, "failed to marshal check result")
	} else {
		fmt.Println(string(b))
	}
}

func printTextResult(targets target.Targets, currentStatus *status.CurrentStatus) {
	for _, t := range targets.GetSortedList() {
		fmt.Printf("%d [%s]\n", t.Version, t.ID)
		for _, app := range t.Apps {
			fmt.Printf("    %-20s%s\n", app.Name, app.URI)
		}
		fmt.Println()
	}
	fmt.Printf("Current version: %s\n", currentStatus.TargetID)
	var areAppsInSync = true
	for _, app := range currentStatus.AppStatuses {
		fmt.Printf("    %-20s%s\n", app.Name, app.URI)
		fmt.Printf("    %-20sfetched:%v; installed:%v; running:%v\n", "", app.Fetched, app.Installed, app.Running)
		fmt.Println()
		if !app.Fetched || !app.Installed || !app.Running {
			areAppsInSync = false
		}
	}
	if currentStatus.TargetID != targets.GetLatestTarget().ID {
		fmt.Printf("New version available: %s. Please run 'fioup update' to get it.\n",
			targets.GetLatestTarget().ID)
	} else if !areAppsInSync {
		fmt.Println("You are running the latest version, but not all apps are in sync." +
			" Please run 'fioup update' to fix this.")
	} else {
		fmt.Println("You are running the latest version.")
	}
}
