// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"context"

	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/state"
)

func Fetch(ctx context.Context, cfg *config.Config, toVersion int, options ...UpdateOpt) error {
	opts := getUpdateOpts(options...)
	return newUpdateRunner([]state.ActionState{
		&state.Check{
			Action:         "fetch",
			UpdateTargets:  false,
			AllowNewUpdate: true,
			Force:          true,
			ToVersion:      toVersion,
			EnableTUF:      opts.EnableTUF,
		},
		&state.Init{},
		&state.Fetch{ProgressHandler: opts.FetchProgressHandler},
	}, updateOptsToRunnerOpt(opts)).Run(ctx, cfg)
}
