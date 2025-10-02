// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/internal/events"
	internal "github.com/foundriesio/fioup/internal/update"
	"github.com/pkg/errors"
)

type (
	Start struct{}
)

var (
	ErrStartFailed = errors.New("start failed")
)

func (s *Start) Name() ActionName { return "Starting" }
func (s *Start) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	if updateCtx.ToTarget.NoApps() {
		fmt.Printf("\t\tno apps to start, prunning current apps\n")
	} else {
		fmt.Printf("\n")
	}
	err := updateCtx.UpdateRunner.Start(ctx)
	if err == nil {
		if err := updateCtx.UpdateRunner.Complete(ctx, update.CompleteWithPruning()); err != nil {
			slog.Debug("failed to complete update with pruning", "error", err)
		}
		updateCtx.Client.UpdateHeaders(updateCtx.ToTarget.AppNames(), updateCtx.ToTarget.ID)
	} else {
		err = fmt.Errorf("%w: %s", ErrStartFailed, err.Error())
	}
	updateCtx.SendEvent(events.InstallationCompleted, err == nil)
	if err := internal.ReportAppsStates(ctx, updateCtx.Client, updateCtx.Config.ComposeConfig()); err != nil {
		slog.Debug("failed to report apps states", "error", err)
	}
	return err
}
