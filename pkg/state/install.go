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

type Install struct{}

func (s *Install) Name() ActionName { return "Installing" }
func (s *Install) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	updateCtx.SendEvent(events.InstallationStarted)
	if updateCtx.ToTarget.NoApps() {
		fmt.Printf("\tstopping all apps, nothing to install\n")
	} else {
		fmt.Printf("\n")
	}
	// Stop apps being updated before installing their updates
	if err := compose.StopApps(ctx, updateCtx.Config.ComposeConfig(), updateCtx.FromTarget.AppURIs()); err != nil {
		return err
	}
	err := updateCtx.UpdateRunner.Install(ctx, compose.WithInstallProgress(update.GetInstallProgressPrinter()))
	// Why not send success/failure info with the event?
	updateCtx.SendEvent(events.InstallationApplied)
	return err
}
