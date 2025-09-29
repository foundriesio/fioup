// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"fmt"
	"time"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/internal/events"
)

type Fetch struct{}

func (s *Fetch) Name() ActionName { return "Fetching" }
func (s *Fetch) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	var err error
	updateState := updateCtx.UpdateRunner.Status().State
	switch updateState {
	case update.StateCreated, update.StateInitializing:
		return fmt.Errorf("update not initialized, cannot fetch")
	case update.StateInitialized, update.StateFetching:
		// We should fast fordward state at the init by default and don't send the events if nothing to download
		updateCtx.SendEvent(events.DownloadStarted)
		if updateCtx.UpdateRunner.Status().TotalBlobsBytes > 0 {
			fmt.Println()
		} else {
			fmt.Println("\t\tskipping, no blobs to download")
		}
		err = updateCtx.UpdateRunner.Fetch(ctx, compose.WithFetchProgress(update.GetFetchProgressPrinter()))
		updateCtx.SendEvent(events.DownloadCompleted, err == nil)
	default:
		if updateCtx.UpdateRunner.Status().TotalBlobsBytes > 0 {
			status := updateCtx.UpdateRunner.Status()
			fmt.Printf("\t\tcompleted at %s\n", status.UpdateTime.Local().Format(time.DateTime))
		} else {
			fmt.Println("\t\tskipping, no blobs to download")
		}
	}
	return err
}
