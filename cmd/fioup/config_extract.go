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
		Use:   "config-extract",
		Short: "Extract the current encrypted configuration to secrets directory",
		Run: func(cmd *cobra.Command, args []string) {
			doConfigExtract(cmd, &opts)
		},
		Args: cobra.NoArgs,
	}
	opts.ApplyToCmd(cmd)
	rootCmd.AddCommand(cmd)
}

func doConfigExtract(_ *cobra.Command, opts *fioconfigOpts) {
	configApp, err := fioconfig.NewAppWithConfig(config.TomlConfig(), opts.secretsDir, opts.unsafeHandlers)
	cobra.CheckErr(err)
	_, err = configApp.Extract()
	cobra.CheckErr(err)
}
