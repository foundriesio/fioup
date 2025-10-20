// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"errors"
	"log/slog"
	"os"
	"strconv"
	"time"

	fioconfig "github.com/foundriesio/fioconfig/app"
	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/pkg/api"
	"github.com/foundriesio/fioup/pkg/client"
	"github.com/foundriesio/fioup/pkg/state"
	"github.com/spf13/cobra"
)

type (
	fioconfigOpts struct {
		secretsDir      string
		unsafeHandlers  bool
		configExtracted bool
	}

	daemonOptions struct {
		runOnce bool

		configEnabled bool
		fioconfig     fioconfigOpts
	}
)

func (opts *fioconfigOpts) ApplyToCmd(cmd *cobra.Command) {
	cmd.Flags().StringVar(&opts.secretsDir, "secrets-dir", "/run/secrets", "Directory to hold FioConfig secrets when enabled.")
	cmd.Flags().BoolVar(&opts.unsafeHandlers, "unsafe-handlers", false, "Enable unsafe FioConfig handlers.")
	_ = cmd.Flags().MarkHidden("unsafe-handlers")
}

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
	cmd.Flags().BoolVar(&opts.configEnabled, "fioconfig", true, "Include FioConfig daemon logic.")
	_ = cmd.Flags().MarkHidden("run-once")
	opts.fioconfig.ApplyToCmd(cmd)
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

	var configApp *fioconfig.App

	if opts.configEnabled {
		configApp, err = fioconfig.NewAppWithConfig(config.TomlConfig(), opts.fioconfig.secretsDir, opts.fioconfig.unsafeHandlers)
		if err != nil {
			slog.Error("Failed to create FioConfig handle", "error", err)
			return
		}
	}

	if eventSender, err = events.NewEventSender(config, gwClient); err != nil {
		slog.Error("Failed to create event sender", "error", err)
		return
	}
	eventSender.Start()
	defer eventSender.Stop()

	for {
		if opts.configEnabled {
			_ = configCheck(&opts.fioconfig, configApp)
		}

		err := api.Update(cmd.Context(), config, -1,
			api.WithGatewayClient(gwClient),
			api.WithEventSender(eventSender),
			api.WithMaxAttempts(3))
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

func configCheck(config *fioconfigOpts, app *fioconfig.App) error {
	if _, err := os.Stat(config.secretsDir); os.IsNotExist(err) {
		slog.Debug("Creating FioConfig secrets directory", "dir", config.secretsDir)
		if err := os.MkdirAll(config.secretsDir, 0o700); err != nil {
			slog.Error("Failed to create secrets directory", "dir", config.secretsDir, "error", err)
			return err
		}
	}
	if !config.configExtracted {
		slog.Debug("Running FioConfig secret extraction")
		if err := app.Extract(); err != nil {
			slog.Error("FioConfig secret extraction failed", "error", err)
			return err
		} else {
			slog.Debug("FioConfig extraction completed successfully")
			config.configExtracted = true
		}
	}
	if err := app.CheckIn(); err != nil {
		if err != fioconfig.NotModifiedError {
			slog.Error("FioConfig check-in failed", "error", err)
			return err
		}
	}
	return nil
}
