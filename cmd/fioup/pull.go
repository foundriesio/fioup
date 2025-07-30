package main

import (
	"github.com/foundriesio/fioup/pkg/fioup"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull the update from the OTA server",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Msg("Starting pull operation...")
		DieNotNil(fioup.Pull(), "failed to pull update from OTA server")
		log.Debug().Msg("Pull operation complete")
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
}
