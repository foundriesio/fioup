// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"fmt"
	"log/slog"
	"regexp"

	fioconfig "github.com/foundriesio/fioconfig/app"
	"github.com/spf13/cobra"
)

type runAndReportOpts struct {
	config       fioconfigOpts
	name         string
	id           string
	artifactsDir string
}

func (opts *runAndReportOpts) ApplyToCmd(cmd *cobra.Command) {
	opts.config.ApplyToCmd(cmd)
	cmd.Flags().StringVar(&opts.name, "name", "", "A short name for the test")
	cmd.Flags().StringVar(&opts.id, "id", "", "UUID for the test")
	cmd.Flags().StringVar(&opts.artifactsDir, "artifacts-dir", "", "Include files from this directory as artifacts in the test result")

	cobra.CheckErr(cobra.MarkFlagRequired(cmd.Flags(), "name"))
}

func init() {
	opts := runAndReportOpts{}
	cmd := &cobra.Command{
		Use:   "run-and-report",
		Short: "Run a command and report the output to the device-gateway",
		Run: func(cmd *cobra.Command, args []string) {
			doRunAndReport(cmd, args, &opts)
		},
		Args: cobra.MinimumNArgs(1),
		Annotations: map[string]string{
			lockFlagKey: "false",
		},
	}
	opts.ApplyToCmd(cmd)
	rootCmd.AddCommand(cmd)
}

func doRunAndReport(_ *cobra.Command, args []string, opts *runAndReportOpts) {
	if len(opts.id) > 0 {
		pattern := `^[A-Za-z0-9\-\_]{15,48}$`
		if !regexp.MustCompile(pattern).MatchString(opts.id) {
			cobra.CheckErr(fmt.Errorf("invalid test ID: %s, must match pattern %s", opts.id, pattern))
		}
	}
	pattern := `^[a-z0-9\-\_]{4,16}$`
	if !regexp.MustCompile(pattern).MatchString(opts.name) {
		cobra.CheckErr(fmt.Errorf("invalid test name: %s, must match pattern %s", opts.name, pattern))
	}

	app, err := fioconfig.NewAppWithConfig(config.TomlConfig(), opts.config.secretsDir, opts.config.unsafeHandlers)
	cobra.CheckErr(err)

	slog.Info("Running command", "args", args)
	cobra.CheckErr(app.RunAndReport(opts.name, opts.id, opts.artifactsDir, args))
}
