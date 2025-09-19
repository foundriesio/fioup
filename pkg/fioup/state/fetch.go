// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"fmt"
	"time"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
)

type Fetch struct{}

func (s *Fetch) Name() StateName { return "Fetching" }
func (s *Fetch) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	var err error
	updateState := updateCtx.UpdateRunner.Status().State
	switch updateState {
	case update.StateCreated, update.StateInitializing:
		return fmt.Errorf("update not initialized, cannot fetch")
	case update.StateInitialized, update.StateFetching:
		fmt.Println()
		err = updateCtx.UpdateRunner.Fetch(ctx, compose.WithFetchProgress(update.GetFetchProgressPrinter()))
	default:
		status := updateCtx.UpdateRunner.Status()
		fmt.Printf("\t\tcompleted at %s\n", status.UpdateTime.Local().Format(time.DateTime))
	}
	return err
}
