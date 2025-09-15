// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package fioup

import (
	"context"
	"fmt"
	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
)

type StagingState struct{}

func (s *StagingState) Name() StateName { return Staging }
func (s *StagingState) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	var err error

	if updateCtx.Runner == nil {
		updateCtx.Runner, err = update.GetCurrentUpdate(updateCtx.Config.ComposeConfig())
		if err != nil {
			if err == update.ErrUpdateNotFound {
				updateCtx.Runner, err = update.NewUpdate(updateCtx.Config.ComposeConfig(), updateCtx.ToTarget.Name())
				if err != nil {
					return fmt.Errorf("failed to create update runner: %w", err)
				}
			} else {
				return fmt.Errorf("failed to get current update: %w", err)
			}
		}
	}

	var apps []string
	state := updateCtx.Runner.Status().State

	if state == update.StateCreated {
		// First time init, stage/init all apps in the target
		apps = updateCtx.ToTarget.Apps()
	}
	if state == update.StateCreated || state == update.StateInitializing {
		fmt.Println()
		err = updateCtx.Runner.Init(ctx, apps, update.WithInitProgress(update.GetInitProgressPrinter()))
		status := updateCtx.Runner.Status()
		fmt.Printf("Diff: %s, %d blobs\n", compose.FormatBytesInt64(status.TotalBlobsBytes), len(status.Blobs))
	} else {
		status := updateCtx.Runner.Status()
		fmt.Printf("\t\t%s, %d blobs\n", compose.FormatBytesInt64(status.TotalBlobsBytes), len(status.Blobs))
	}
	return err
}
