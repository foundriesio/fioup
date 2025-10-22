// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/internal/events"
)

type Install struct{}

func (s *Install) Name() ActionName { return "Installing" }
func (s *Install) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	updateCtx.SendEvent(events.InstallationStarted)
	currentState := updateCtx.UpdateRunner.Status().State
	if currentState == update.StateStarted || currentState == update.StateStarting {
		return nil
	}
	// Stop apps being updated before installing their updates
	if err := compose.StopApps(ctx, updateCtx.Config.ComposeConfig(), updateCtx.FromTarget.AppURIs()); err != nil {
		return err
	}
	err := updateCtx.UpdateRunner.Install(ctx)
	updateCtx.SendEvent(events.InstallationApplied)
	return err
}
