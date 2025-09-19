// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package fioup

import (
	"context"

	"github.com/foundriesio/fioup/pkg/fioup/config"
	"github.com/foundriesio/fioup/pkg/fioup/state"
)

func Fetch(ctx context.Context, cfg *config.Config, toVersion int) error {
	return state.NewUpdateRunner([]state.ActionState{
		&state.Check{
			Action:         "fetch",
			UpdateTargets:  true,
			AllowNewUpdate: true,
			ToVersion:      toVersion,
		},
		&state.Init{},
		&state.Fetch{},
	}).Run(ctx, cfg)
}
