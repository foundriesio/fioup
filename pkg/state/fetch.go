// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"fmt"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/internal/events"
)

type Fetch struct {
	ProgressHandler compose.FetchProgressFunc
}

func (s *Fetch) Name() ActionName { return "Fetching" }
func (s *Fetch) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	var err error
	updateState := updateCtx.UpdateRunner.Status().State
	switch updateState {
	case update.StateCreated, update.StateInitializing:
		return fmt.Errorf("update not initialized, cannot fetch")
	case update.StateInitialized, update.StateFetching:
		updateCtx.SendEvent(events.DownloadStarted)
		err = updateCtx.UpdateRunner.Fetch(ctx, compose.WithFetchProgress(s.ProgressHandler))
		updateCtx.SendEvent(events.DownloadCompleted, err)
	default:
		updateCtx.AlreadyFetched = true
	}
	return err
}
