package main

import (
	"github.com/foundriesio/fioup/internal/update"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	opts := update.UpdateOptions{}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start execution of the updated target. A install operation must be performed first.",
		Run: func(cmd *cobra.Command, args []string) {
			doRun(&opts)
		},
		Args: cobra.NoArgs,
	}
	rootCmd.AddCommand(cmd)
}

func doRun(opts *update.UpdateOptions) {
	opts.DoRun = true
	err := update.Update(config, opts)
	DieNotNil(err, "Failed to perform update")
	log.Info().Msgf("Run operation complete")
}
