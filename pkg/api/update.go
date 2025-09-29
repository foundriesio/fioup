// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"context"

	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/state"
	"github.com/pkg/errors"
)

func Update(ctx context.Context, cfg *config.Config, toVersion int) error {
	updateRunner := state.NewUpdateRunner([]state.ActionState{
		&state.Check{
			Action:         "update",
			UpdateTargets:  true,
			AllowNewUpdate: true,
			ToVersion:      toVersion,
		},
		&state.Init{},
		&state.Fetch{},
		&state.Install{},
		&state.Start{},
	})
	err := updateRunner.Run(ctx, cfg)
	if !errors.Is(err, state.ErrStartFailed) {
		return err
	}
	// if app failed to start, do rollback
	return state.NewUpdateRunner([]state.ActionState{
		&state.Check{
			Action:         "rollback",
			UpdateTargets:  false,
			AllowNewUpdate: true,
			ToVersion:      updateRunner.GetFromTarget().Version,
		},
		&state.Init{CheckState: true},
		&state.Fetch{},
		&state.Install{},
		&state.Start{},
	}).Run(ctx, cfg)
}
