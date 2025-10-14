// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"context"

	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/state"
)

func Update(ctx context.Context, cfg *config.Config, toVersion int, skipIfRunning bool) error {
	return state.NewUpdateRunner([]state.ActionState{
		&state.Check{
			Action:         "update",
			UpdateTargets:  true,
			AllowNewUpdate: true,
			SkipIfRunning:  skipIfRunning,
			ToVersion:      toVersion,
		},
		&state.Init{},
		&state.Fetch{},
		&state.Install{},
		&state.Start{},
	}).Run(ctx, cfg)
}
