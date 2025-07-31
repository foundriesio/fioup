package main

import (
	"github.com/foundriesio/fioup/internal/register"
	"github.com/spf13/cobra"
)

func init() {

	var opt register.RegisterOptions
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register device with Foundries.io",
		Run: func(cmd *cobra.Command, args []string) {
			doRegister(&opt)
		},
	}
	cmd.Flags().BoolVar(&opt.UseServer, "use-ostree-server", true, "Use OSTree Proxy server instead of the Device Gateway to pull the ostree repo.")
	cmd.Flags().BoolVar(&opt.Production, "production", false, "Mark the device as a production device.")
	// cmd.Flags().BoolVar(&opt.StartDaemon, "start-daemon", true, "Start the "+SOTA_CLIENT+" systemd service after registration.")
	cmd.Flags().StringVar(&opt.SotaDir, "sota-dir", register.SOTA_DIR, "The directory to install to keys and configuration to.")
	cmd.Flags().StringVar(&opt.DeviceGroup, "device-group", "", "Assign this device to a device group.")
	cmd.Flags().StringVar(&opt.Factory, "factory", "", "The factory name to subscribe to.")
	// cmd.Flags().StringVar(&opt.HsmSoPin, "hsm-so-pin", "", "The PKCS#11 security officer pin - HSM only.")
	cmd.Flags().BoolVar(&opt.Vuuid, "validate-uuid", true, "Validate the UUID before calling the remote service.")
	cmd.Flags().StringVar(&opt.Hwid, "hwid", register.HARDWARE_ID, "The hardware identifier for the device type.")
	cmd.Flags().StringVar(&opt.PacmanTags, "tags", "", "Configure "+register.SOTA_CLIENT+" to only apply updates from Targets with these tags.")
	cmd.Flags().StringVar(&opt.ApiToken, "api-token", "", "API token for authentication. If not provided, oauth2 will be used instead.")
	// cmd.Flags().StringVar(&opt.HsmModule, "hsm-module", "", "The PKCS#11 implementation (.so library) - HSM only.")
	// cmd.Flags().StringVar(&opt.HsmPin, "hsm-pin", "", "The PKCS#11 pin - HSM only.")
	cmd.Flags().StringVar(&opt.UUID, "uuid", "", "A per-device UUID. If not provided, one will be generated.")
	cmd.Flags().StringVar(&opt.Name, "name", "", "The name of the device as it should appear in the dashboard. When not specified, the device's UUID will be used instead.")
	cmd.Flags().StringVar(&opt.ApiTokenHeader, "api-token-header", "OSF-TOKEN", "The HTTP header to use for authentication.")
	cmd.Flags().BoolVar(&opt.Force, "force", false, "Force registration, removing data from previous execution.")

	rootCmd.AddCommand(cmd)
}

func doRegister(opts *register.RegisterOptions) {
	err := register.RegisterDevice(opts)
	DieNotNil(err, "Failed to register device")
}
