// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/foundriesio/fioup/internal/register"
	"github.com/spf13/cobra"
)

func init() {

	var opt register.RegisterOptions
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register device with Foundries.io",
		Run: func(cmd *cobra.Command, args []string) {
			var err error
			opt.SotaDir, err = filepath.Abs(opt.SotaDir)
			cobra.CheckErr(err)
			doRegister(&opt)
		},
	}
	cmd.Flags().BoolVar(&opt.Production, "production", false, "Mark the device as a production device.")
	cmd.Flags().StringVar(&opt.SotaDir, "sota-dir", register.SOTA_DIR, "The directory to install to keys and configuration to.")
	cmd.Flags().StringVar(&opt.DeviceGroup, "device-group", "", "Assign this device to a device group.")
	cmd.Flags().StringVar(&opt.Factory, "factory", "", "The factory name to subscribe to.")
	cmd.Flags().StringVar(&opt.PacmanTag, "tag", "main", "Configure "+register.SOTA_CLIENT+" to only apply updates from Targets with this tag.")
	cmd.Flags().StringVar(&opt.ApiToken, "api-token", "", "API token for authentication. If not provided, oauth2 will be used instead.")
	cmd.Flags().StringVar(&opt.UUID, "uuid", "", "A per-device UUID. If not provided, one will be generated.")
	cmd.Flags().StringVar(&opt.Name, "name", "", "The name of the device as it should appear in the dashboard. When not specified, the device's UUID will be used instead.")
	cmd.Flags().StringVar(&opt.ApiTokenHeader, "api-token-header", "OSF-TOKEN", "The HTTP header to use for authentication.")
	cmd.Flags().BoolVar(&opt.Force, "force", false, "Force registration, removing data from previous execution.")
	cmd.Flags().StringVar(&opt.HardwareID, "hw-id", register.HARDWARE_ID, "Hardware ID to assign to this device.")

	cobra.CheckErr(cmd.Flags().MarkHidden("api-token-header"))
	cobra.CheckErr(cmd.Flags().MarkHidden("api-token"))
	cobra.CheckErr(cmd.Flags().MarkHidden("device-group"))
	cobra.CheckErr(cmd.Flags().MarkHidden("production"))
	cobra.CheckErr(cmd.Flags().MarkHidden("uuid"))
	cobra.CheckErr(cmd.Flags().MarkHidden("hw-id"))

	rootCmd.AddCommand(cmd)
}

func doRegister(opts *register.RegisterOptions) {
	h := oauthHandler{}
	err := register.RegisterDevice(opts, &h)
	if err != nil && errors.Is(err, os.ErrExist) {
		fmt.Printf("ERROR: Device already registered under %s. ", opts.SotaDir)
		fmt.Println("Re-run with `--force 1` to remove existing registration data.")
		os.Exit(1)
	}
	cobra.CheckErr(err)
	fmt.Printf("Device %s is now registered at factory %s\n", opts.Name, opts.Factory)
}

type oauthHandler struct {
	i int
}

func (oauthHandler) ShowAuthInfo(deviceName, userCode, url string, expiresMinutes int) {
	fmt.Printf("Visit the link below in your browser to authorize this new device. This link will expire in %d minutes.\n", expiresMinutes)
	fmt.Println("  Device UUID:", deviceName)
	fmt.Println("  User code:", userCode)
	fmt.Println("  Browser URL:", url)
	fmt.Println()
}

func (h *oauthHandler) Tick() {
	wheels := []rune{'|', '/', '-', '\\'}
	fmt.Printf("Waiting for authorization %c\r", wheels[h.i%len(wheels)])
	h.i++
}
