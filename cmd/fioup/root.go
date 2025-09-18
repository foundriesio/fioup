// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"github.com/foundriesio/fioup/internal/update"
	fioup "github.com/foundriesio/fioup/pkg/fioup/config"
	"os"
	"strings"

	"github.com/foundriesio/fioconfig/sotatoml"
	"github.com/moby/term"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	verbose     bool
	configPaths []string
	config      *sotatoml.AppConfig
	config1     *fioup.Config

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
			config1, err = fioup.NewConfig(config)
			DieNotNil(err, "failed to load configuration")
			DieNotNil(update.InitializeDatabase(config1.GetDBPath()), "failed to initialize database")
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
