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
		Use:   "update <target_name_or_version>",
		Short: "Update TUF metadata, download and install the selected target",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				opts.TargetId = args[0]
			}
			doUpdate(&opts)
		},
		Args: cobra.RangeArgs(0, 1),
	}

	cmd.Flags().StringVarP(&opts.SrcDir, "src-dir", "s", "", "Directory that contains an offline update bundle.")
	cmd.Flags().BoolVar(&opts.EnableTuf, "tuf", false, "Enable TUF metadata checking, instead of reading targets.json directly.")

	rootCmd.AddCommand(cmd)
}

func doUpdate(opts *update.UpdateOptions) {
	opts.DoCheck = true
	opts.DoPull = true
	opts.DoInstall = true
	opts.DoRun = true
	err := update.Update(config, opts)
	DieNotNil(err, "Failed to perform update")
	log.Info().Msgf("Update operation complete")
}
