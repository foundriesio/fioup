// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package states

import (
	"context"
	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/pkg/fioup/config"
	"github.com/foundriesio/fioup/pkg/fioup/target"
)

// StateName represents all possible states (both action and status)
type StateName string

const (
	Checking   StateName = "Checking"
	Checked    StateName = "Checked"
	Staging    StateName = "Staging"
	Staged     StateName = "Staged"
	Fetching   StateName = "Fetching"
	Fetched    StateName = "Fetched"
	Installing StateName = "Installing"
	Installed  StateName = "Installed"
	Starting   StateName = "Starting"
	Started    StateName = "Started"
)

type (
	// State interface for all states
	State interface {
		Name() StateName
		Execute(ctx context.Context, updateCtx *UpdateContext) error
	}

	// UpdateContext holds the state machine context
	UpdateContext struct {
		Config         *config.Config
		TargetProvider target.TargetProvider
		FromTarget     target.Target
		ToVersion      int
		ToTarget       target.Target
		CurrentState   StateName
		Runner         update.Runner
	}
)
