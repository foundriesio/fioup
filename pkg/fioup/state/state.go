// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/pkg/fioup/config"
	"github.com/foundriesio/fioup/pkg/fioup/target"
)

// StateName represents all possible states (both action and status)
type (
	StateName  string
	ActionType string
)

const (
	ActionTypeNewUpdate    ActionType = "states:actiontype:new-update"
	ActionTypeResumeUpdate ActionType = "states:actiontype:resume-update"
)

type (
	// ActionState interface for all states
	ActionState interface {
		Name() StateName
		Execute(ctx context.Context, updateCtx *UpdateContext) error
	}

	// UpdateContext holds the state machine context
	UpdateContext struct {
		Config *config.Config

		FromTarget   target.Target
		ToTarget     target.Target
		UpdateRunner update.Runner

		CurrentState StateName
	}
)
