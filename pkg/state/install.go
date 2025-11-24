// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/internal/events"
)

type Install struct {
	ProgressHandler compose.InstallProgressFunc
}

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
		// Installation successful, send applied event, which indicates installation is done successfully
		updateCtx.SendEvent(events.InstallationApplied)
	} else {
		// Installation failed, send completed event with error
		updateCtx.SendEvent(events.InstallationCompleted, err)
	}
	return err
}
