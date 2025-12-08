// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"fmt"

	"github.com/foundriesio/fioup/pkg/api"
	"github.com/spf13/cobra"
)

func init() {
	opts := commonOptions{}
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Update TUF metadata",
		Run: func(cmd *cobra.Command, args []string) {
			doCheck(cmd, &opts)
		},
		Args: cobra.NoArgs,
		Annotations: map[string]string{
			lockFlagKey: "true",
		},
	}
	addCommonOptions(cmd, &opts)
	rootCmd.AddCommand(cmd)
}

func doCheck(cmd *cobra.Command, opts *commonOptions) {
	targets, currentStatus, err := api.Check(cmd.Context(), config, api.WithTUF(opts.enableTuf))
	DieNotNil(err, "failed to check for updates")
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
