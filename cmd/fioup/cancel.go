package main

import (
	"github.com/foundriesio/fioup/internal/update"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	opts := update.UpdateOptions{
		SrcDir:    "",
		EnableTuf: false,
	}

	cmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel the current update operation",
		Run: func(cmd *cobra.Command, args []string) {
			doCancel(&opts)
		},
		Args: cobra.NoArgs,
	}
	rootCmd.AddCommand(cmd)
}

func doCancel(opts *update.UpdateOptions) {
	err := update.CancelPendingUpdate(config, opts)
	DieNotNil(err, "Failed to perform cancel")
	log.Info().Msgf("Cancel operation complete")
}
