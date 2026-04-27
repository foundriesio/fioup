// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"encoding/json"
	"fmt"

	"github.com/foundriesio/fioup/pkg/api"
	"github.com/foundriesio/fioup/pkg/state"
	"github.com/foundriesio/fioup/pkg/status"
	"github.com/foundriesio/fioup/pkg/target"
	"github.com/spf13/cobra"
)

type (
	checkOptions struct {
		commonOptions
		Format string
	}
)

func init() {
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
			doCheck(cmd, &opts)
		default:
			return fmt.Errorf("invalid value for --format: %s (must be text or json)", opts.Format)
		}
		return nil
	}
	addCommonOptions(cmd, &opts.commonOptions)
	rootCmd.AddCommand(cmd)
}

func doCheck(cmd *cobra.Command, opts *checkOptions) {
	targets, currentStatus, err := api.Check(cmd.Context(), config, api.WithTUF(opts.enableTuf))
	DieNotNil(err, "failed to check for updates")

	if opts.Format == "json" {
		printJsonResult(targets, currentStatus)
	} else {
		printTextResult(targets, currentStatus)
	}
}

type (
	CheckResultUpdate struct {
		SelectedTarget any    `json:"selected_target,omitempty"`
		Type           string `json:"type,omitempty"`
		Description    string `json:"description,omitempty"`
	}

	CheckResult struct {
		AppsAreInSync  bool              `json:"apps_are_in_sync"`
		UpdateRequired bool              `json:"update_required"`
		Update         CheckResultUpdate `json:"update,omitempty"`
	}
)

func printJsonResult(targets target.Targets, currentStatus *status.CurrentStatus) {
	var areAppsInSync = true
	for _, app := range currentStatus.AppStatuses {
		if !app.Fetched || !app.Installed || !app.Running {
			areAppsInSync = false
		}
	}

	description := ""
	var updateType state.UpdateType
	var selectedTarget target.Target
	updateRequired := false
	if currentStatus.TargetID != targets.GetLatestTarget().ID {
		description = fmt.Sprintf("New version available: %s", targets.GetLatestTarget().ID)
		updateRequired = true
		selectedTarget = targets.GetLatestTarget()
		updateType = state.UpdateTypeUpdate
	} else if !areAppsInSync {
		description = "You are running the latest version, but not all apps are in sync."
		updateRequired = true
		selectedTarget = targets.GetLatestTarget()
		updateType = state.UpdateTypeSync
	}

	result := struct {
		Targets       target.Targets        `json:"targets"`
		CurrentStatus *status.CurrentStatus `json:"current_status"`
		CheckResult   CheckResult           `json:"check_result"`
	}{
		Targets:       targets,
		CurrentStatus: currentStatus,
		CheckResult: CheckResult{
			AppsAreInSync:  areAppsInSync,
			UpdateRequired: updateRequired,
		},
	}

	if updateRequired {
		result.CheckResult.Update = CheckResultUpdate{
			SelectedTarget: struct {
				ID      string `json:"id"`
				Version int    `json:"version"`
			}{
				ID:      selectedTarget.ID,
				Version: selectedTarget.Version,
			},
			Type:        string(updateType),
			Description: description,
		}
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
	currentTarget := targets.GetTargetByID(currentStatus.TargetID)

	fmt.Println("Current version:", currentTarget.Version)
	if currentTarget.ID != targets.GetLatestTarget().ID {
		fmt.Println("Latest version: ", targets.GetLatestTarget().Version)
		fmt.Println("Status:          Update available")
	} else if currentStatus.AreAppsHealthy() {
		fmt.Println("Status:          Up-to-date")
	} else {
		fmt.Println("Status:          Up-to-date (degraded)")
		fmt.Println("Unhealthy apps:")
		for _, app := range currentStatus.AppStatuses {
			if app.Running && app.Installed && app.Fetched {
				continue
			}
			fmt.Printf(" - %s\n", app.Name)
			fmt.Printf("   %s\n", app.URI)
			fmt.Printf("   state: fetched:%v; installed:%v; running:%v\n", app.Fetched, app.Installed, app.Running)
			fmt.Println()
		}
	}
}
