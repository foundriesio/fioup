// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	fioconfig "github.com/foundriesio/fioconfig/app"
	"github.com/foundriesio/fioconfig/sotatoml"
	cfg "github.com/foundriesio/fioup/pkg/config"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

const (
	lockFileName = "fioup.lock"
	lockFlagKey  = "lock-flag"
)

var (
	verbose     bool
	configPaths []string
	config      *cfg.Config
	lockFile    *os.File

	rootCmd = &cobra.Command{
		Use:   "fioup",
		Short: "Utility to perform OTA Updates managed by FoundriesFactory (c)",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			var logLevel slog.Level
			// Set global log level based on verbose flag
			if verbose {
				logLevel = slog.LevelDebug
			} else {
				logLevel = slog.LevelInfo
			}

			var handler slog.Handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: logLevel,
			})
			if isatty.IsTerminal(os.Stderr.Fd()) {
				handler = fioconfig.NewConsoleHandler(handler, os.Stdout, os.Stderr)
			}

			logger := slog.New(handler)
			slog.SetDefault(logger)
			if cmd.Name() != "register" && cmd.Name() != "version" {
				// Load configuration unless the "register" command is invoked
				var err error
				config, err = cfg.NewConfig(configPaths)
				cobra.CheckErr(err)
			}
			// If the lock flag is set, then acquire lock to prevent concurrent executions from different processes
			if l := cmd.Annotations[lockFlagKey]; l == "true" {
				cobra.CheckErr(acquireLock())
			}
		},
	}
)

func Execute() error {
	overrides := [][2]string{
		{"FIOUP_CFG_DIRS", "cfg-dirs"},
		{"FIOUP_VERBOSE", "verbose"},
	}
	for _, override := range overrides {
		val := os.Getenv(override[0])
		if len(val) > 0 {
			flag := rootCmd.PersistentFlags().Lookup(override[1])
			cobra.CheckErr(flag.Value.Set(val))
		}
	}

	if strings.HasSuffix(os.Args[0], "docker-credential-fioup") {
		if len(os.Args) != 2 || os.Args[1] != "get" {
			fmt.Printf("Usage: %s get\n", os.Args[0])
			os.Exit(1)
		}
		rootCmd.PersistentPreRun(rootCmd, nil)
		os.Exit(runDockerCredsHelper())
	}
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose debug logging")
	rootCmd.PersistentFlags().StringSliceVarP(&configPaths, "cfg-dirs", "c",
		sotatoml.DEF_CONFIG_ORDER, "A comma-separated list of paths to search for .toml configuration files")
}

func acquireLock() error {
	var err error
	lockFilePath := filepath.Join(runtimeLockDir(), lockFileName)
	lockFile, err = os.OpenFile(lockFilePath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("failed to open or create lock file: %w", err)
	}
	// Try to acquire exclusive lock, non-blocking
	if err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("another instance of fioup is already running: %w", err)
	}
	slog.Debug("successfully acquired lock file", "path", lockFilePath)
	return nil
}

func runtimeLockDir() string {
	// If running as root (including via sudo), use /run
	if os.Geteuid() == 0 {
		slog.Debug("Running as root, using /run for runtime lock directory")
		if _, err := os.Stat("/run"); err == nil {
			return "/run"
		}
	}
	// Non-root: prefer XDG_RUNTIME_DIR
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	// Non-root fallback: /run/user/$UID
	uid := os.Geteuid()
	userRun := filepath.Join("/run/user", strconv.Itoa(uid))
	if _, err := os.Stat(userRun); err == nil {
		return userRun
	}
	// Last resort
	return os.TempDir()
}
