package main

import (
	"github.com/foundriesio/fioup/internal/update"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	opts := update.UpdateOptions{}
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Update TUF metadata",
		Run: func(cmd *cobra.Command, args []string) {
			doCheck(&opts)
		},
		Args: cobra.NoArgs,
	}
	addCommonOptions(cmd, &opts)
	rootCmd.AddCommand(cmd)
}

func doCheck(opts *update.UpdateOptions) {
	opts.DoCheck = true
	err := update.Update(config, opts)
	DieNotNil(err, "Failed to perform check operation")
	log.Info().Msgf("Check operation complete")
}
