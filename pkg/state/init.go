// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/internal/events"
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
		updateCtx.SendEvent(events.UpdateInitStarted)
		err = updateCtx.UpdateRunner.Init(ctx, apps,
			update.WithInitAllowEmptyAppList(true),
			update.WithInitCheckStatus(s.CheckState))
		updateCtx.SendEvent(events.UpdateInitCompleted, err)
	}
	status := updateCtx.UpdateRunner.Status()
	updateCtx.Size.Bytes, updateCtx.Size.Blobs = status.TotalBlobsBytes, len(status.Blobs)
	updateCtx.FetchedAt = updateCtx.UpdateRunner.Status().FetchTime
	return err
}
