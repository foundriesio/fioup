// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/pkg/client"
	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/target"
)

type (
	// ActionName Name of the state action
	ActionName string
	// ActionState interface for all states
	ActionState interface {
		Name() ActionName
		Execute(ctx context.Context, updateCtx *UpdateContext) error
	}

	UpdateMode string

	Diff struct {
		ToFetch struct {
			Bytes int64
			Blobs int
		}
	}

	UpdateInfo struct {
		Mode          UpdateMode
		FromTarget    target.Target
		ToTarget      target.Target
		CurrentAction ActionName
		// current state number in the state machine
		CurrentStateNum int
		// total number of states in the state machine
		TotalStates int

		Diff Diff
	}

	// UpdateContext holds the state machine context
	UpdateContext struct {
		UpdateInfo
		Config *config.Config

		EventSender *events.EventSender
		Client      *client.GatewayClient
		PreHandler  PreStateActionHandler
		PostHandler PostStateActionHandler

		UpdateRunner update.Runner
	}
)

const (
	UpdateModeNew    = "new"
	UpdateModeResume = "resume"
	UpdateModeSync   = "sync"
)
