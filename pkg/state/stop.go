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
)

type Stop struct{}

var (
	ErrStopAppsFailed = fmt.Errorf("stopping apps failed")
)

func (s *Stop) Name() ActionName { return "Stopping" }
func (s *Stop) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	currentState := updateCtx.UpdateRunner.Status().State
	if currentState.IsOneOf(update.StateInstalled, update.StateStarting, update.StateStarted, update.StateCompleting) {
		// No need to stop apps if the ongoing update is already in installing, installed, starting, or started state
		return nil
	}
	if !currentState.IsOneOf(update.StateFetched, update.StateInstalling) {
		return fmt.Errorf("%w: invalid state %s for stopping apps; must be in state %s or %s", ErrInvalidOngoingUpdate,
			currentState, update.StateFetched, update.StateInstalling)
	}
	// Installation starts from stopping the currently running apps that are being updated
	updateCtx.SendEvent(events.InstallationStarted)
	// Stop apps being updated before installing their updates
	err := compose.StopApps(ctx, updateCtx.Config.ComposeConfig(), updateCtx.FromTarget.AppURIs())
	if err != nil {
		// If stopping apps failed, it means that update has completed with failure, so send InstallationCompleted event with failure
		if currentStatus, errStatus := status.GetCurrentStatus(ctx, updateCtx.Config.ComposeConfig()); errStatus == nil {
			updateCtx.CurrentStatus = currentStatus
		} else {
			slog.Error("failed to get current app statuses after stop failure", "error", errStatus)
		}
		updateCtx.SendEvent(events.InstallationCompleted, err)
		err = fmt.Errorf("%w: %w", ErrStopAppsFailed, err)
	}
	return err
}
