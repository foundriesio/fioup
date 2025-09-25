//go:build disable_main

// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage:", os.Args[0], "manpages|bash-completion <path>")
		return
	}
	tool := os.Args[1]
	dst := os.Args[2]

	switch tool {
	case "manpages":
		if err := os.Mkdir(dst, 0o750); err != nil && !errors.Is(err, os.ErrExist) {
			cobra.CheckErr(err)
		}
		header := &doc.GenManHeader{
			Title:   "FIOUP",
			Section: "1",
		}
		cobra.CheckErr(doc.GenManTree(rootCmd, header, dst))
	case "bash-completion":
		cobra.CheckErr(rootCmd.GenBashCompletionFile(dst))
	default:
		cobra.CheckErr(fmt.Errorf("invalid tool name: %s", tool))
	}
}
