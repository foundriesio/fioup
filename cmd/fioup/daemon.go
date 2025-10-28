// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/foundriesio/composeapp/pkg/update"
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

func (opts daemonOptions) initAPIs() (*client.GatewayClient, *events.EventSender) {
	gw, err := client.NewGatewayClient(config, nil, "")
	DieNotNil(err, "Failed to create gateway client")

	var sender *events.EventSender

	if !opts.runOnce {
		sender, err = events.NewEventSender(config, gw)
		DieNotNil(err, "Failed to create event sender")
	}

	return gw, sender
}

func (opts daemonOptions) pollingInterval() time.Duration {
	pollingSecStr := config.TomlConfig().GetDefault("uptane.polling_seconds", "300")
	pollingSec, err := strconv.Atoi(pollingSecStr)
	if err != nil || pollingSec <= 0 {
		pollingSec = 300
		slog.Warn("Invalid value for uptane.polling_seconds. Using default value", "value", pollingSecStr, "default", pollingSec)
	}
	return time.Duration(time.Duration(pollingSec) * time.Second)
}

func doDaemon(cmd *cobra.Command, opts *daemonOptions) {
	interval := opts.pollingInterval()
	ctx := cmd.Context()

	gwClient, eventSender := opts.initAPIs()

	if eventSender != nil {
		eventSender.Start()
		defer eventSender.Stop()
	}

	for {
		if nowait := updateCheck(cmd.Context(), opts, gwClient, eventSender); nowait {
			continue
		}
		slog.Info("Waiting before next check...", "interval", interval)
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

func updateCheck(ctx context.Context, opts *daemonOptions, gwClient *client.GatewayClient, eventSender *events.EventSender) (nowait bool) {
	err := api.Update(ctx, config, -1,
		api.WithGatewayClient(gwClient),
		api.WithEventSender(eventSender),
		api.WithRequireLatest(true),
		api.WithMaxAttempts(3),
		api.WithPreStateHandler(preStateHandler),
		api.WithPostStateHandler(postStateHandler),
		api.WithFetchProgressHandler(update.GetFetchProgressPrinter(update.WithIndentation(8))),
		api.WithInstallProgressHandler(update.GetInstallProgressPrinter(update.WithIndentation(8))),
		api.WithStartProgressHandler(appStartHandler))
	if err != nil && errors.Is(err, state.ErrNewerVersionIsAvailable) {
		slog.Info("Cancelling current update, going to start a new one for the newer version")
		_, err := api.Cancel(ctx, config)
		if err != nil {
			slog.Error("Error canceling old update", "error", err)
		} else {
			// If cancelation was successful, proceed without waiting
			nowait = true
		}
	} else if err != nil && errors.Is(err, state.ErrStartFailed) {
		slog.Info("Error starting updated target", "error", err)
		if !opts.runOnce {
			// Retry installation, or do a sync update, without waiting
			// If runOnce is set, exit execution in the return statement bellow
			nowait = true
		}
	} else if err != nil && !errors.Is(err, state.ErrCheckNoUpdate) {
		slog.Error("Error during update", "error", err)
	}
	if opts.runOnce {
		slog.Debug("Run once mode, exiting")
		DieNotNil(err)
		os.Exit(0)
	}
	return
}
