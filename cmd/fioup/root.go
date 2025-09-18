// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"os"
	"path"
	"strings"

	"github.com/foundriesio/composeapp/pkg/compose"
	v1 "github.com/foundriesio/composeapp/pkg/compose/v1"
	"github.com/foundriesio/fioconfig/sotatoml"
	"github.com/moby/term"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	verbose     bool
	configPaths []string
	// TODO: introduce a notion of fioup config that encapsulates these two types of configs
	config        *sotatoml.AppConfig
	composeConfig *compose.Config

	rootCmd = &cobra.Command{
		Use:   "fioup",
		Short: "Utility to perform OTA Updates managed by FoundriesFactory (c)",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Set global log level based on verbose flag
			if verbose {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			} else {
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
			}

			// Output pretty console if terminal (optional)
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: !term.IsTerminal(uintptr(os.Stderr.Fd()))})

			var err error
			config, err = sotatoml.NewAppConfig(configPaths)
			DieNotNil(err, "failed to load configuration from paths: "+strings.Join(configPaths, ", "))
			composeConfig, err = getComposeConfig(config)
			DieNotNil(err, "failed to get compose configuration")
		},
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose debug logging")
	rootCmd.PersistentFlags().StringSliceVarP(&configPaths, "cfg-dirs", "c",
		sotatoml.DEF_CONFIG_ORDER, "A comma-separated list of paths to search for .toml configuration files")
}

func getComposeConfig(config *sotatoml.AppConfig) (*compose.Config, error) {
	cfg, err := v1.NewDefaultConfig(
		v1.WithStoreRoot(config.GetDefault("pacman.reset_apps_root", "/var/sota/reset-apps")),
		v1.WithComposeRoot(config.GetDefault("pacman.compose_apps_root", "/var/sota/compose-apps")),
		v1.WithUpdateDB(path.Join(config.GetDefault("storage.path", "/var/sota"), "updates.db")),
	)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
