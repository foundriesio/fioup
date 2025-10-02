package main

import (
	"encoding/json"
	"fmt"

	"github.com/foundriesio/fioup/pkg/client"
	"github.com/spf13/cobra"
)

var whoamiFormat string

func init() {
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Fetch and display device information as known by the device gateway",
		RunE: func(cmd *cobra.Command, args []string) error {
			if whoamiFormat != "json" && whoamiFormat != "text" {
				return fmt.Errorf("invalid value for --format: %s (must be text or json)", whoamiFormat)
			}
			doWhoAmI()
			return nil
		},
		Args: cobra.NoArgs,
	}
	cmd.Flags().StringVar(&whoamiFormat, "format", "text", "Format the output. Values: [text | json]")
	rootCmd.AddCommand(cmd)
}

func doWhoAmI() {
	client, err := client.NewGatewayClient(config, nil, "")
	cobra.CheckErr(err)
	self, err := client.Self()
	DieNotNil(err)
	cobra.CheckErr(err)
	if whoamiFormat == "text" {
		fmt.Println("Name:     ", self.Name)
		fmt.Println("Uuuid:    ", self.Uuid)
		fmt.Println("Tag:      ", self.Tag)
		fmt.Println("Last seen:", self.LastSeen.AsTime())
		fmt.Println("Created:  ", self.CreatedAt.AsTime())
		fmt.Println("Public key:")
		fmt.Println(self.PubKey)
	} else {
		selfBytes, err := json.MarshalIndent(self, "", "  ")
		cobra.CheckErr(err)
		fmt.Println(string(selfBytes))
	}
}
