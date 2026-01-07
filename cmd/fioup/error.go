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
	state.ErrInvalidActionForState:   10,
	state.ErrMetaUpdateFailed:        20,
	state.ErrNoMatchingTarget:        21,
	state.ErrTargetNotFound:          22,
	state.ErrInvalidOngoingUpdate:    23,
	state.ErrNoUpdateInProgress:      24,
	state.ErrCheckNoUpdate:           25,
	state.ErrNewerVersionIsAvailable: 26,
	state.ErrInitFailed:              30,
	state.ErrFetchFailed:             40,
	state.ErrFetchNoSpace:            41,
	state.ErrStopAppsFailed:          50,
	state.ErrInstallFailed:           60,
	state.ErrStartFailed:             70,
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
