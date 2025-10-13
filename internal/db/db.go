// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package db

import (
	"fmt"

	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/internal/targets"
)

func InitializeDatabase(dbFilePath string) error {
	err := targets.CreateTargetsTable(dbFilePath)
	if err != nil {
		return fmt.Errorf("failed to create targets table %w", err)
	}

	err = events.CreateEventsTable(dbFilePath)
	if err != nil {
		return fmt.Errorf("failed to create events table %w", err)
	}

	return nil
}
