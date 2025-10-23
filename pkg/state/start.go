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
	err := updateCtx.UpdateRunner.Start(ctx, compose.WithStartProgressHandler(s.ProgressHandler))
	if err == nil {
		if err := updateCtx.UpdateRunner.Complete(ctx, update.CompleteWithPruning()); err != nil {
			slog.Debug("failed to complete update with pruning", "error", err)
		}
		updateCtx.Client.UpdateHeaders(updateCtx.ToTarget.AppNames(), updateCtx.ToTarget.ID)
	} else {
		err = fmt.Errorf("%w: %s", ErrStartFailed, err.Error())
	}
	updateCtx.SendEvent(events.InstallationCompleted, err == nil)
	return err
}
