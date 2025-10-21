// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"context"

	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/state"
)

func Fetch(ctx context.Context, cfg *config.Config, toVersion int) error {
	return state.NewUpdateRunner([]state.ActionState{
		&state.Check{
			Action:         "fetch",
			UpdateTargets:  false,
			AllowNewUpdate: true,
			Force:          true,
			ToVersion:      toVersion,
		},
		&state.Init{},
		&state.Fetch{},
	}).Run(ctx, cfg)
}
