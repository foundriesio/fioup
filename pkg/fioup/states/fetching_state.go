// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package states

import (
	"context"
	"fmt"
	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
	"time"
)

type FetchingState struct{}

func (s *FetchingState) Name() StateName { return Fetching }
func (s *FetchingState) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	var err error
	updateState := updateCtx.Runner.Status().State
	switch updateState {
	case update.StateCreated, update.StateInitializing:
		return fmt.Errorf("update not initialized, cannot fetch")
	case update.StateInitialized, update.StateFetching:
		fmt.Println()
		err = updateCtx.Runner.Fetch(ctx, compose.WithFetchProgress(update.GetFetchProgressPrinter()))
	default:
		status := updateCtx.Runner.Status()
		fmt.Printf("\t\tcompleted at %s\n", status.UpdateTime.Local().Format(time.DateTime))
	}
	return err
}
