package main

import (
	"github.com/foundriesio/fioup/internal/update"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	opts := update.UpdateOptions{}
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the update. A pull operation must be performed first.",
		Run: func(cmd *cobra.Command, args []string) {
			doInstall(&opts)
		},
		Args: cobra.NoArgs,
	}
	rootCmd.AddCommand(cmd)
}

func doInstall(opts *update.UpdateOptions) {
	opts.DoInstall = true
	err := update.Update(config, opts)
	DieNotNil(err, "Failed to perform install operation")
	log.Info().Msgf("Install operation complete")
}
