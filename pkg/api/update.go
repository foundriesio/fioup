// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"context"

	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/state"
)

type (
	UpdateOpts struct {
		Force bool
	}
	UpdateOpt func(*UpdateOpts)
)

func WithForceUpdate(enabled bool) UpdateOpt {
	return func(o *UpdateOpts) {
		o.Force = enabled
	}
}

func Update(ctx context.Context, cfg *config.Config, toVersion int, options ...UpdateOpt) error {
	opts := &UpdateOpts{
		Force: false,
	}
	for _, o := range options {
		o(opts)
	}
	return state.NewUpdateRunner([]state.ActionState{
		&state.Check{
			Action:         "update",
			UpdateTargets:  true,
			AllowNewUpdate: true,
			Force:          opts.Force,
			ToVersion:      toVersion,
		},
		&state.Init{},
		&state.Fetch{},
		&state.Install{},
		&state.Start{},
	}).Run(ctx, cfg)
}
