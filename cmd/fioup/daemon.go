// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/foundriesio/fioup/pkg/api"
	"github.com/foundriesio/fioup/pkg/state"
	"github.com/spf13/cobra"
)

type (
	daemonOptions struct {
		runOnce bool
	}
)

func init() {
	opts := daemonOptions{
		runOnce: false,
	}
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Start the update agent daemon",
		Run: func(cmd *cobra.Command, args []string) {
			doDaemon(cmd, &opts)
		},
		Args: cobra.NoArgs,
	}
	cmd.Flags().BoolVar(&opts.runOnce, "run-once", false, "Run a single update check and exit.")
	_ = cmd.Flags().MarkHidden("run-once")
	rootCmd.AddCommand(cmd)
}

func doDaemon(cmd *cobra.Command, opts *daemonOptions) {
	pollingSecStr := config.TomlConfig().GetDefault("uptane.polling_seconds", "300")
	pollingSec, err := strconv.Atoi(pollingSecStr)
	if err != nil || pollingSec <= 0 {
		pollingSec = 300
		slog.Debug("Invalid value for uptane.polling_seconds. Using default value", "value", pollingSecStr, "default", pollingSec)
	}
	interval := time.Duration(time.Duration(pollingSec) * time.Second)
	ctx := cmd.Context()
	for {
		err := api.Update(cmd.Context(), config, -1, api.WithMaxAttempts(3))
		if err != nil && !errors.Is(err, state.ErrCheckNoUpdate) {
			slog.Error("Error during update", "error", err)
		}
		if opts.runOnce {
			slog.Debug("Run once mode, exiting")
			return
		}
		slog.Info("Waiting before next check...", "interval", interval)
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}
