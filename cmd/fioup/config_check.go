// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	fioconfig "github.com/foundriesio/fioconfig/app"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type fioconfigOpts struct {
	secretsDir      string
	unsafeHandlers  bool
	configExtracted bool
}

func (opts *fioconfigOpts) ApplyToCmd(cmd *cobra.Command) {
	cmd.Flags().StringVar(&opts.secretsDir, "secrets-dir", "/run/secrets", "Directory to hold fioconfig secrets when enabled.")
	cmd.Flags().BoolVar(&opts.unsafeHandlers, "unsafe-handlers", false, "Enable unsafe fioconfig handlers.")
	_ = cmd.Flags().MarkHidden("unsafe-handlers")
}

// CanExtract ensures the `secretsDir` exists, and the process has permission to create files in it
func (opts fioconfigOpts) AssertCanExtract() {
	if len(opts.secretsDir) == 0 {
		// This shouldn't be possible. Just defensive coding
		DieNotNil(errors.New("`secrets-dir` not configured"))
	}
	if _, err := os.Stat(opts.secretsDir); os.IsNotExist(err) {
		slog.Debug("Creating fioconfig secrets directory", "dir", opts.secretsDir)
		err = os.MkdirAll(opts.secretsDir, 0o700)
		DieNotNil(err, fmt.Sprintf("Failed to create secrets directory `%s`", opts.secretsDir))
	}
	testfile := filepath.Join(opts.secretsDir, ".test-writeable")
	err := os.WriteFile(testfile, nil, 0o740)
	DieNotNil(err, "Unable to create files in `secrets-dir`:")
	_ = os.Remove(testfile)
}

func init() {
	opts := fioconfigOpts{}
	cmd := &cobra.Command{
		Use:   "config-check",
		Short: "Check for config updates",
		Run: func(cmd *cobra.Command, args []string) {
			doCheckConfig(cmd, &opts)
		},
		Args: cobra.NoArgs,
	}
	opts.ApplyToCmd(cmd)
	rootCmd.AddCommand(cmd)
}

func doCheckConfig(_ *cobra.Command, opts *fioconfigOpts) {
	// the aklite on-change handler can send a SIGHUP that we can ignore
	// when not running as a daemon
	sigHUP := make(chan os.Signal, 1)
	signal.Notify(sigHUP, syscall.SIGHUP)

	configApp, err := fioconfig.NewAppWithConfig(config.TomlConfig(), opts.secretsDir, opts.unsafeHandlers)
	cobra.CheckErr(err)
	_, err = configCheck(opts, configApp)
	if !errors.Is(err, fioconfig.NotModifiedError) {
		cobra.CheckErr(err)
	}
}

func configCheck(config *fioconfigOpts, app *fioconfig.App) (changed bool, err error) {
	config.AssertCanExtract()
	if !config.configExtracted {
		slog.Debug("Running fioconfig secret extraction")
		if changed, err = app.Extract(); err != nil {
			slog.Error("fioconfig secret extraction failed", "error", err)
			return
		} else {
			slog.Debug("fioconfig extraction completed successfully")
			config.configExtracted = true
		}
	}
	if changed, err = app.CheckIn(); err != nil {
		if err != fioconfig.NotModifiedError {
			slog.Error("Fioconfig check-in failed", "error", err)
			return
		}
	}
	return
}
