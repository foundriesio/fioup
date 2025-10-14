// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"fmt"
	"os"
)

// DieNotNil logs the error and exits with code 1.
func DieNotNil(err error, message ...string) {
	DieNotNilWithCode(err, 1, message...)
}

// DieNotNilWithCode logs the error and exits with the given code.
func DieNotNilWithCode(err error, exitCode int, message ...string) {
	if err != nil {
		parts := []interface{}{"ERROR:"}
		for _, p := range message {
			parts = append(parts, p)
		}
		parts = append(parts, err)
		fmt.Println(parts...)
		os.Exit(exitCode)
	}
}
