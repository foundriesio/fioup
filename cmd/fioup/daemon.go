// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/pkg/api"
	"github.com/foundriesio/fioup/pkg/client"
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
	var gwClient *client.GatewayClient
	var eventSender *events.EventSender
	if gwClient, err = client.NewGatewayClient(config, nil, ""); err != nil {
		slog.Error("Failed to create gateway client", "error", err)
		return
	}
	if eventSender, err = events.NewEventSender(config, gwClient); err != nil {
		slog.Error("Failed to create event sender", "error", err)
		return
	}
	eventSender.Start()
	defer eventSender.Stop()

	for {
		err := api.Update(cmd.Context(), config, -1,
			api.WithGatewayClient(gwClient),
			api.WithEventSender(eventSender),
			api.WithRequireLatest(true),
			api.WithMaxAttempts(3),
			api.WithPreStateHandler(beforeStateHandler),
			api.WithPostStateHandler(afterStateHandler))
		if err != nil && errors.Is(err, state.ErrNewerVersionIsAvailable) {
			slog.Info("Cancelling current update, going to start a new one for the newer version")
			_, err := api.Cancel(cmd.Context(), config)
			if err != nil {
				slog.Error("Error canceling old update", "error", err)
			} else {
				// If cancelation was successful, proceed without waiting
				continue
			}
		} else if err != nil && !errors.Is(err, state.ErrCheckNoUpdate) {
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

func beforeStateHandler(state api.StateName, update *api.UpdateInfo) {
	slog.Info("Entering", "state", state)
}

func afterStateHandler(state api.StateName, update *api.UpdateInfo) {
	slog.Info("Exiting", "state", state)
}
