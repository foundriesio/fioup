// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/pkg/api"
	"github.com/foundriesio/fioup/pkg/client"
	cfg "github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/state"
	"github.com/spf13/cobra"
)

type (
	daemonOptions struct {
		runOnce bool
	}

	updater struct {
		opts   *daemonOptions
		gw     *client.GatewayClient
		sender *events.EventSender

		sleepInterval time.Duration
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

func NewUpdater(opts *daemonOptions) *updater {
	u := updater{
		opts: opts,
	}
	u.reload(false)
	return &u
}

func (u *updater) reload(reloadConfig bool) {
	var err error

	u.Close()

	if reloadConfig {
		config, err = cfg.NewConfig(configPaths)
		DieNotNil(err)
	}

	u.gw, err = client.NewGatewayClient(config, nil, "")
	DieNotNil(err, "Failed to create gateway client")

	if !u.opts.runOnce {
		u.sender, err = events.NewEventSender(config, u.gw)
		DieNotNil(err, "Failed to create event sender")
		u.sender.Start()
	}

	pollingSecStr := config.TomlConfig().GetDefault("uptane.polling_seconds", "300")
	pollingSec, err := strconv.Atoi(pollingSecStr)
	if err != nil || pollingSec <= 0 {
		pollingSec = 300
		slog.Warn("Invalid value for uptane.polling_seconds. Using default value", "value", pollingSecStr, "default", pollingSec)
	}

	u.sleepInterval = time.Duration(time.Duration(pollingSec) * time.Second)
}

func doDaemon(cmd *cobra.Command, opts *daemonOptions) {
	slog.Info("Daemon starting", "pid", os.Getpid())
	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer cancel()

	sigHUP := make(chan os.Signal, 1)
	signal.Notify(sigHUP, syscall.SIGHUP)

	updater := NewUpdater(opts)
	defer updater.Close()

	for {
		if nowait := updater.checkUpdates(ctx); nowait {
			continue
		}

		if reloadConfig := updater.sleep(ctx, sigHUP); reloadConfig {
			updater.reload(true)
		}
	}
}

func (u updater) Close() {
	if u.sender != nil {
		u.sender.Stop()
	}
}

func (u updater) sleep(ctx context.Context, sigHUP chan os.Signal) (reloadConfig bool) {
	if u.sleepInterval > 0 {
		slog.Info("Waiting before next check...", "interval", u.sleepInterval)
	}
	select {
	case <-ctx.Done():
		slog.Debug("Received SIGINT, exiting")
		u.Close()
		os.Exit(0)
	case <-sigHUP:
		slog.Info("Received SIGHUP")
		reloadConfig = true
	case <-time.After(u.sleepInterval):
	}
	return
}

func (u updater) checkUpdates(ctx context.Context) (nowait bool) {
	err := api.Update(ctx, config, -1,
		api.WithGatewayClient(u.gw),
		api.WithEventSender(u.sender),
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
		if !u.opts.runOnce {
			// Retry installation, or do a sync update, without waiting
			// If runOnce is set, exit execution in the return statement bellow
			nowait = true
		}
	} else if err != nil && !errors.Is(err, state.ErrCheckNoUpdate) {
		slog.Error("Error during update", "error", err)
	}
	if u.opts.runOnce {
		slog.Debug("Run once mode, exiting")
		DieNotNil(err)
		os.Exit(0)
	}
	return
}
