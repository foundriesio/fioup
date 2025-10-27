// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"context"
	"fmt"

	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/state"
	"github.com/foundriesio/fioup/pkg/status"
	"github.com/foundriesio/fioup/pkg/target"
	"github.com/pkg/errors"
)

func Check(ctx context.Context, cfg *config.Config, options ...UpdateOpt) (target.Targets, *status.CurrentStatus, error) {
	opts := getUpdateOpts(options...)
	updateRunner := newUpdateRunner([]state.ActionState{
		&state.Check{
			Action:         "update",
			UpdateTargets:  true,
			AllowNewUpdate: true,
			Force:          true,
			ToVersion:      -1,
			SyncCurrent:    opts.SyncCurrent,
			RequireLatest:  opts.RequireLatest,
			MaxAttempts:    opts.MaxAttempts,
			EnableTUF:      opts.EnableTUF,
		},
	}, updateOptsToRunnerOpt(opts))
	if err := updateRunner.Run(ctx, cfg); err != nil && !errors.Is(err, state.ErrCheckNoUpdate) {
		return nil, nil, err
	}
	currentStatus := updateRunner.ctx.CurrentStatus
	if currentStatus == nil {
		if s, err := status.GetCurrentStatus(ctx, cfg.ComposeConfig()); err == nil {
			currentStatus = s
		} else {
			return nil, nil, fmt.Errorf("failed to get current status: %w", err)
		}
	}
	return updateRunner.ctx.Targets, currentStatus, nil
}
