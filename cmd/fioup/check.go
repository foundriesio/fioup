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
		Use:   "check",
		Short: "Update TUF metadata",
		Run: func(cmd *cobra.Command, args []string) {
			doCheck(&opts)
		},
		Args: cobra.NoArgs,
	}

	cmd.Flags().StringVarP(&opts.SrcDir, "src-dir", "s", "", "Directory that contains an offline update bundle.")
	cmd.Flags().BoolVar(&opts.EnableTuf, "tuf", false, "Enable TUF metadata checking, instead of reading targets.json directly.")

	rootCmd.AddCommand(cmd)
}

func doCheck(opts *update.UpdateOptions) {
	opts.DoCheck = true
	err := update.Update(config, opts)
	DieNotNil(err, "Failed to perform update")
	log.Info().Msgf("Check operation complete")
}
