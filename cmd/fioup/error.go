package main

import (
	"os"

	"github.com/rs/zerolog/log"
)

// DieNotNil logs the error and exits with code 1.
func DieNotNil(err error, message ...string) {
	DieNotNilWithCode(err, 1, message...)
}

// DieNotNilWithCode logs the error and exits with the given code.
func DieNotNilWithCode(err error, exitCode int, message ...string) {
	if err != nil {
		event := log.Error().Err(err)
		for _, m := range message {
			event = event.Str("msg", m)
		}
		event.Msg("fatal error")
		os.Exit(exitCode)
	}
}
