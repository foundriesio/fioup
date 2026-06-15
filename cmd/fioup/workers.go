// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	defaultFetchWorkers = 3
	minFetchWorkers     = 1
	maxFetchWorkers     = 10
)

// addWorkersFlag registers the --workers flag on a blob-fetching command.
func addWorkersFlag(cmd *cobra.Command, workers *int) {
	cmd.Flags().IntVar(workers, "workers", defaultFetchWorkers,
		fmt.Sprintf("Number of concurrent blob-download workers (%d-%d).", minFetchWorkers, maxFetchWorkers))
}

// validateWorkers aborts the command if the requested worker count is out of range.
func validateWorkers(workers int) {
	if workers < minFetchWorkers || workers > maxFetchWorkers {
		DieNotNil(fmt.Errorf("--workers must be between %d and %d, got %d",
			minFetchWorkers, maxFetchWorkers, workers))
	}
}
