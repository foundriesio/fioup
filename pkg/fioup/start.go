// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package fioup

import (
	"context"
	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/pkg/fioup/config"
	"github.com/foundriesio/fioup/pkg/fioup/state"
)

func Start(ctx context.Context, cfg *config.Config) error {
	return state.NewUpdateRunner([]state.ActionState{
		&state.Check{
			Action:         "start",
			UpdateTargets:  false,
			AllowNewUpdate: false,
			AllowedStates: []update.State{
				update.StateInstalled,
				update.StateStarting,
			},
		},
		&state.Init{},
		&state.Fetch{},
		&state.Install{},
		&state.Start{},
	}).Run(ctx, cfg)
}
