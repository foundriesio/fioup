// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"context"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/state"
)

func Start(ctx context.Context, cfg *config.Config, options ...UpdateOpt) error {
	opts := getUpdateOpts(options...)
	return newUpdateRunner([]state.ActionState{
		&state.Check{
			Action:         "start",
			UpdateTargets:  false,
			AllowNewUpdate: false,
			AllowedStates: []update.State{
				update.StateInstalled,
				update.StateStarting,
				update.StateStarted,
				update.StateCompleting,
			},
			ToVersion: -1,
			EnableTUF: opts.EnableTUF,
		},
		&state.Init{},
		&state.Fetch{},
		&state.Stop{},
		&state.Install{ProgressHandler: opts.InstallProgressHandler},
		&state.Start{ProgressHandler: opts.StartProgressHandler},
	}, updateOptsToRunnerOpt(opts)).Run(ctx, cfg)
}
