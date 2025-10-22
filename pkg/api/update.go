// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"context"

	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/pkg/client"
	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/state"
)

type (
	UpdateOpts struct {
		UpdateRunnerOpts
		Force         bool
		SyncCurrent   bool
		MaxAttempts   int
		RequireLatest bool
	}
	UpdateOpt func(*UpdateOpts)
)

func WithPreStateHandler(handler PreStateHandler) UpdateOpt {
	return func(o *UpdateOpts) {
		o.PreStateHandler = handler
	}
}

func WithPostStateHandler(handler PostStateHandler) UpdateOpt {
	return func(o *UpdateOpts) {
		o.PostStateHandler = handler
	}
}

func WithForceUpdate(enabled bool) UpdateOpt {
	return func(o *UpdateOpts) {
		o.Force = enabled
	}
}

func WithSyncCurrent(enabled bool) UpdateOpt {
	return func(o *UpdateOpts) {
		o.SyncCurrent = enabled
	}
}

func WithRequireLatest(enabled bool) UpdateOpt {
	return func(o *UpdateOpts) {
		o.RequireLatest = enabled
	}
}

func WithMaxAttempts(count int) UpdateOpt {
	return func(o *UpdateOpts) {
		o.MaxAttempts = count
	}
}

func WithEventSender(sender *events.EventSender) UpdateOpt {
	return func(o *UpdateOpts) {
		o.EventSender = sender
	}
}

func WithGatewayClient(client *client.GatewayClient) UpdateOpt {
	return func(o *UpdateOpts) {
		o.GatewayClient = client
	}
}

func Update(ctx context.Context, cfg *config.Config, toVersion int, options ...UpdateOpt) error {
	opts := getUpdateOpts(options...)
	return newUpdateRunner([]state.ActionState{
		&state.Check{
			Action:         "update",
			UpdateTargets:  true,
			AllowNewUpdate: true,
			Force:          opts.Force,
			ToVersion:      toVersion,
			SyncCurrent:    opts.SyncCurrent,
			RequireLatest:  opts.RequireLatest,
			MaxAttempts:    opts.MaxAttempts,
		},
		&state.Init{},
		&state.Fetch{},
		&state.Stop{},
		&state.Install{},
		&state.Start{},
	}, updateOptsToRunnerOpt(opts)).Run(ctx, cfg)
}

func getUpdateOpts(options ...UpdateOpt) *UpdateOpts {
	opts := &UpdateOpts{}
	for _, o := range options {
		o(opts)
	}
	return opts
}

func updateOptsToRunnerOpt(opts *UpdateOpts) UpdateRunnerOpt {
	return func(r *UpdateRunnerOpts) {
		r.EventSender = opts.EventSender
		r.GatewayClient = opts.GatewayClient
		r.PreStateHandler = opts.PreStateHandler
		r.PostStateHandler = opts.PostStateHandler
	}
}
