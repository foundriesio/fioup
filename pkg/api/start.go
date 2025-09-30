// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"context"
	"fmt"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/state"
	"github.com/pkg/errors"
)

func Start(ctx context.Context, cfg *config.Config) error {
	updateRunner := state.NewUpdateRunner([]state.ActionState{
		&state.Check{
			Action:         "start",
			UpdateTargets:  false,
			AllowNewUpdate: false,
			AllowedStates: []update.State{
				update.StateInstalled,
				update.StateStarting,
			},
			ToVersion: -1,
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
	fmt.Printf("Update failed, rolling back to previous version; err: %s\n", err.Error())
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
