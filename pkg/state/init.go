// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"fmt"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
)

type Init struct {
	CheckState bool
}

func (s *Init) Name() ActionName { return "Initializing" }
func (s *Init) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	var err error
	if updateCtx.UpdateRunner == nil {
		updateCtx.UpdateRunner, err = update.NewUpdate(updateCtx.Config.ComposeConfig(), updateCtx.ToTarget.ID)
	}

	var apps []string
	state := updateCtx.UpdateRunner.Status().State

	if state == update.StateCreated {
		// First time init, stage/init all apps in the target
		apps = updateCtx.ToTarget.AppURIs()
	}
	if state == update.StateCreated || state == update.StateInitializing {
		if !updateCtx.ToTarget.NoApps() {
			fmt.Println()
		}
		err = updateCtx.UpdateRunner.Init(ctx, apps,
			update.WithInitProgress(update.GetInitProgressPrinter()),
			update.WithInitAllowEmptyAppList(true),
			update.WithInitCheckStatus(s.CheckState))
		status := updateCtx.UpdateRunner.Status()
		if !updateCtx.ToTarget.NoApps() {
			fmt.Printf("Diff: %s, %d blobs\n", compose.FormatBytesInt64(status.TotalBlobsBytes), len(status.Blobs))
		} else {
			fmt.Printf("\tdiff: %s, %d blobs\n", compose.FormatBytesInt64(status.TotalBlobsBytes), len(status.Blobs))
		}
	} else {
		status := updateCtx.UpdateRunner.Status()
		fmt.Printf("\t%s, %d blobs\n", compose.FormatBytesInt64(status.TotalBlobsBytes), len(status.Blobs))
	}
	return err
}
