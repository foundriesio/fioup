// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	fioconfig "github.com/foundriesio/fioconfig/app"
	"github.com/spf13/cobra"
)

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
	configApp, err := fioconfig.NewAppWithConfig(config.TomlConfig(), opts.secretsDir, opts.unsafeHandlers)
	cobra.CheckErr(err)
	cobra.CheckErr(configCheck(opts, configApp))
}
