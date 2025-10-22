// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"fmt"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/internal/events"
)

type Stop struct{}

func (s *Stop) Name() ActionName { return "Stopping" }
func (s *Stop) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	// Installation starts from stopping the currently running apps that are being updated
	updateCtx.SendEvent(events.InstallationStarted)
	currentState := updateCtx.UpdateRunner.Status().State
	if currentState.IsOneOf(update.StateInstalled, update.StateStarting, update.StateStarted) {
		// No need to stop apps if the ongoing update is already in installing, installed, starting, or started state
		return nil
	}
	if !currentState.IsOneOf(update.StateFetched, update.StateInstalling) {
		return fmt.Errorf("invalid state %s for stopping apps; must be in state %s or %s", currentState,
			update.StateFetched, update.StateInstalling)
	}
	// Stop apps being updated before installing their updates
	return compose.StopApps(ctx, updateCtx.Config.ComposeConfig(), updateCtx.FromTarget.AppURIs())
}
