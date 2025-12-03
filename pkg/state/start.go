// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/pkg/status"
	"github.com/pkg/errors"
)

type (
	Start struct {
		ProgressHandler compose.AppStartProgress
	}
)

var (
	ErrStartFailed = errors.New("start failed")
)

func (s *Start) Name() ActionName { return "Starting" }
func (s *Start) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	var err error
	currentState := updateCtx.UpdateRunner.Status().State
	if !currentState.IsOneOf(update.StateInstalled, update.StateStarting, update.StateStarted, update.StateCompleting) {
		return fmt.Errorf("cannot start or complete update in state %s", currentState.String())
	}
	if currentState.IsOneOf(update.StateInstalled, update.StateStarting) {
		err = updateCtx.UpdateRunner.Start(ctx, compose.WithStartProgressHandler(s.ProgressHandler))
	}
	if err == nil {
		updateCtx.completeUpdate(ctx)
		updateCtx.Client.UpdateHeaders(updateCtx.ToTarget.AppNames(), updateCtx.ToTarget.ID)
	} else {
		err = fmt.Errorf("%w: %s", ErrStartFailed, err.Error())
	}
	if currentStatus, errStatus := status.GetCurrentStatus(ctx, updateCtx.Config.ComposeConfig()); errStatus == nil {
		updateCtx.CurrentStatus = currentStatus
	} else {
		slog.Error("failed to get current app statuses update completion", "error", errStatus)
	}

	// Update storage usage info after update completion to reflect actual usage
	if err := updateCtx.getAndSetStorageUsageInfo(); err != nil {
		slog.Debug("failed to get storage usage info after fetch", "error", err)
	}
	updateCtx.SendEvent(events.InstallationCompleted, err)
	return err
}

func (u *UpdateContext) completeUpdate(ctx context.Context) {
	var err error
	// 1. First attempt with pruning
	if err = u.UpdateRunner.Complete(ctx, update.CompleteWithPruning()); err == nil {
		return
	}
	slog.Debug("update completion with pruning failed; retrying", "error", err)
	// 2. Second attempt with pruning
	if err = u.UpdateRunner.Complete(ctx, update.CompleteWithPruning()); err == nil {
		return
	}
	slog.Error("update completion with pruning failed; trying without pruning", "error", err)
	// 3. Fallback to complete without pruning
	if err = u.UpdateRunner.Complete(ctx); err == nil {
		slog.Warn("completed update without pruning; some dangling blobs may remain")
		return
	}
	// 4. Final attempt without pruning and force
	if err = u.UpdateRunner.Complete(ctx, update.CompleteWithForce()); err == nil {
		slog.Warn("completed update without pruning and with force; some dangling blobs may remain")
		return
	}
	// 4. Total failure
	slog.Error(
		"failed to complete update after the app successfully started; some dangling blobs may remain",
		"error", err,
	)
}
