// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"context"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/state"
)

func Install(ctx context.Context, cfg *config.Config) error {
	return state.NewUpdateRunner([]state.ActionState{
		&state.Check{
			Action:         "install",
			UpdateTargets:  false,
			AllowNewUpdate: false,
			AllowedStates: []update.State{
				update.StateFetched,
				update.StateInstalling,
			},
			ToVersion: -1,
		},
		&state.Init{},
		&state.Fetch{},
		&state.Install{},
	}).Run(ctx, cfg)
}
