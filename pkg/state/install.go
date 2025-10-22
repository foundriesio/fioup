// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/internal/events"
)

type Install struct{}

func (s *Install) Name() ActionName { return "Installing" }
func (s *Install) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	currentState := updateCtx.UpdateRunner.Status().State
	if currentState.IsOneOf(update.StateStarting, update.StateStarted) {
		// No need to install updates if the ongoing update is already in starting or started state
		return nil
	}
	err := updateCtx.UpdateRunner.Install(ctx)
	updateCtx.SendEvent(events.InstallationApplied)
	return err
}
