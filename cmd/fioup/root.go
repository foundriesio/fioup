package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
)

var (
	verbose bool
	rootCmd = &cobra.Command{
		Use:   "fioup",
		Short: "Utility to perform OTA Updates managed by FoundriesFactory (c)",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Set global log level based on verbose flag
			if verbose {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			} else {
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
			}
			// Output pretty console if terminal (optional)
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		},
		Run: func(cmd *cobra.Command, args []string) {
			DieNotNil(cmd.Help())
		},
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose debug logging")

	// Override usage template
	rootCmd.SetUsageTemplate(`{{.UseLine}}

{{.Short}}

Usage:
  {{.CommandPath}} [global flags] <command> [command flags] [arguments...]

Available Commands:
{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}

Global flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

Use "{{.CommandPath}} [command] --help" for more information about a command.
`)
}
