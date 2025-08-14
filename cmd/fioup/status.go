package main

import (
	"github.com/foundriesio/fioup/internal/update"
	"github.com/spf13/cobra"
)

func init() {
	opts := update.UpdateOptions{}
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show information about the running target and pending update status, if any",
		Run: func(cmd *cobra.Command, args []string) {
			doStatus(&opts)
		},
		Args: cobra.NoArgs,
	}
	rootCmd.AddCommand(cmd)
}

func doStatus(opts *update.UpdateOptions) {
	err := update.Status(config, opts)
	DieNotNil(err, "Failed to get status infomation")
}
