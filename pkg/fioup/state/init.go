// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"fmt"
	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
)

type Init struct{}

func (s *Init) Name() StateName { return "Initializing" }
func (s *Init) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	var err error
	if updateCtx.UpdateRunner == nil {
		updateCtx.UpdateRunner, err = update.NewUpdate(updateCtx.Config.ComposeConfig(), updateCtx.ToTarget.Name())
	}

	var apps []string
	state := updateCtx.UpdateRunner.Status().State

	if state == update.StateCreated {
		// First time init, stage/init all apps in the target
		apps = updateCtx.ToTarget.Apps()
	}
	if state == update.StateCreated || state == update.StateInitializing {
		fmt.Println()
		err = updateCtx.UpdateRunner.Init(ctx, apps, update.WithInitProgress(update.GetInitProgressPrinter()))
		status := updateCtx.UpdateRunner.Status()
		fmt.Printf("Diff: %s, %d blobs\n", compose.FormatBytesInt64(status.TotalBlobsBytes), len(status.Blobs))
	} else {
		status := updateCtx.UpdateRunner.Status()
		fmt.Printf("\t\t%s, %d blobs\n", compose.FormatBytesInt64(status.TotalBlobsBytes), len(status.Blobs))
	}
	return err
}
