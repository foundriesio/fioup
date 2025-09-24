// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/foundriesio/fioconfig/sotatoml"
	cfg "github.com/foundriesio/fioup/pkg/fioup/config"
	"github.com/moby/term"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	verbose     bool
	configPaths []string
	config      *cfg.Config

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
			config, err = cfg.NewConfig(configPaths)
			cobra.CheckErr(err)
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
