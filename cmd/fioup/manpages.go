//go:build disable_main

// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage:", os.Args[0], "<path for manpages>")
		return
	}
	header := &doc.GenManHeader{
		Title:   "FIOUP",
		Section: "1",
	}
	cobra.CheckErr(doc.GenManTree(rootCmd, header, os.Args[1]))
}
