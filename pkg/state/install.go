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

type Install struct {
	ProgressHandler compose.InstallProgressFunc
}

var (
	ErrInstallFailed = errors.New("installation failed")
)

func (s *Install) Name() ActionName { return "Installing" }
func (s *Install) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	currentState := updateCtx.UpdateRunner.Status().State
	if currentState.IsOneOf(update.StateInstalled, update.StateStarting, update.StateStarted, update.StateCompleting) {
		// No need to install updates if the ongoing update is already in installed, starting or started state
		return nil
	}
	var opts []compose.InstallOption
	if s.ProgressHandler != nil {
		opts = append(opts, compose.WithInstallProgress(s.ProgressHandler))
	}
	err := updateCtx.UpdateRunner.Install(ctx, opts...)
	if err == nil {
		updateCtx.SendEvent(events.InstallationApplied)
	} else {
		// If installation failed, it means that update has completed with failure, so send InstallationCompleted event with failure
		if currentStatus, errStatus := status.GetCurrentStatus(ctx, updateCtx.Config.ComposeConfig()); errStatus == nil {
			updateCtx.CurrentStatus = currentStatus
		} else {
			slog.Error("failed to get current app statuses after install failure", "error", errStatus)
		}
		updateCtx.SendEvent(events.InstallationCompleted, err)
		err = fmt.Errorf("%w: %w", ErrInstallFailed, err)
	}
	return err
}
