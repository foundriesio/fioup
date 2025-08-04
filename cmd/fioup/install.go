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
		Use:   "install",
		Short: "Install the update. A pull operation must be performed first.",
		Run: func(cmd *cobra.Command, args []string) {
			doInstall(&opts)
		},
		Args: cobra.NoArgs,
	}

	cmd.Flags().StringVarP(&opts.SrcDir, "src-dir", "s", "", "Directory that contains an offline update bundle.")
	cmd.Flags().BoolVar(&opts.EnableTuf, "tuf", false, "Enable TUF metadata checking, instead of reading targets.json directly.")

	rootCmd.AddCommand(cmd)
}

func doInstall(opts *update.UpdateOptions) {
	opts.DoInstall = true
	err := update.Update(config, opts)
	DieNotNil(err, "Failed to perform update")
	log.Info().Msgf("Install operation complete")
}
