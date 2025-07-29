package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = cobra.Command{
	Use:   "fioup",
	Short: "Utility to perform OTA Updates managed by FoundriesFactory (c)",
}

func Execute() error {
	return rootCmd.Execute()
}
