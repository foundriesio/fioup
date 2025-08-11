package main

import (
	"github.com/foundriesio/fioup/internal/update"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	opts := update.UpdateOptions{}
	cmd := &cobra.Command{
		Use:   "pull <target_name_or_version>",
		Short: "Pull the update from the OTA server",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				opts.TargetId = args[0]
			}
			doPull(&opts)
		},
		Args: cobra.RangeArgs(0, 1),
	}
	addCommonOptions(cmd, &opts)
	rootCmd.AddCommand(cmd)
}

func doPull(opts *update.UpdateOptions) {
	opts.DoPull = true
	err := update.Update(config, opts)
	DieNotNil(err, "Failed to perform update")
	log.Info().Msgf("Pull operation complete")
}
