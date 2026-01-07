// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"fmt"
	"os"

	"github.com/foundriesio/fioup/pkg/state"
	"github.com/pkg/errors"
)

var errorsMap = map[error]int{
	state.ErrCheckNoUpdate:           0,
	state.ErrNewerVersionIsAvailable: 0,
	state.ErrCheckInFailed:           4,
	state.ErrNoMatchingTarget:        6,
	state.ErrTargetNotFound:          20,
	state.ErrInvalidOngoingUpdate:    30,
	state.ErrNoOngoingUpdate:         40,
	state.ErrDownloadFailed:          50,
	state.ErrDownloadFailedNoSpace:   60,
	state.ErrInstallFailed:           150,
	state.ErrStartFailed:             160,
	state.ErrStopAppsFailed:          170,
}

func errorToExitCode(err error) int {
	var exitCode int
	if err != nil {
		exitCode = 1
		for mappedErr, code := range errorsMap {
			if errors.Is(err, mappedErr) {
				exitCode = code
				break
			}
		}
	}
	return exitCode
}

// DieNotNil logs the error and exits with code 1.
func DieNotNil(err error, message ...string) {
	DieNotNilWithCode(err, errorToExitCode(err), message...)
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
