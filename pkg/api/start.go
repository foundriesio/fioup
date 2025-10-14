// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"context"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/state"
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
	return updateRunner.Run(ctx, cfg)
}
