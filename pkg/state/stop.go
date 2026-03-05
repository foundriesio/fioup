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
	"github.com/foundriesio/fioup/pkg/target"
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
		return fmt.Errorf("%w: invalid state %s for stopping apps; must be in state %s or %s", ErrInvalidActionForState,
			currentState, update.StateFetched, update.StateInstalling)
	}
	// Installation starts from stopping of the required apps
	updateCtx.SendEvent(events.InstallationStarted)

	var appsToStop target.Apps
	if updateCtx.Type == UpdateTypeUpdate || updateCtx.IsForcedUpdate {
		// Stop all apps if it is a target/version update or a forced update.
		appsToStop = updateCtx.FromTarget.Apps
	} else {
		// If it is a sync non-forced update, only stop the apps that are being removed or
		// those in sync list that are not "healthy"(running/installed/fetched).
		appsToStop = append(appsToStop, updateCtx.AppDiff.Remove...)
		for _, app := range updateCtx.AppDiff.Sync {
			if updateCtx.CurrentStatus == nil {
				// Handle edge case if the Stop state is executed without running the "Check" state that populates the CurrentStatus
				slog.Warn("current apps' status is not available in update context, " +
					"will stop app in sync list without checking their status")
				appsToStop = append(appsToStop, app)
				continue
			}
			appStatus, ok := updateCtx.CurrentStatus.AppStatuses[app.URI]
			if !ok {
				appsToStop = append(appsToStop, app)
				slog.Warn("app in sync list not found in current status", "appURI", app.URI)
				continue
			}
			if !appStatus.Running || !appStatus.Installed || !appStatus.Fetched {
				appsToStop = append(appsToStop, app)
				continue
			}
		}
	}
	slog.Debug("apps to stop", "updateType", updateCtx.Type, "isForcedUpdate", updateCtx.IsForcedUpdate, "apps", appsToStop.Names())
	var err error
	if len(appsToStop) > 0 {
		// Invoke compose.StopApps only when there are apps to stop, as compose.StopApps will stop all apps if empty list is passed in,
		// which is not the intended behavior here.
		err = compose.StopApps(ctx, updateCtx.Config.ComposeConfig(), appsToStop.URIs())
	}
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
