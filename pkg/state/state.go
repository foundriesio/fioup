// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"log/slog"

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

	// UpdateContext holds the state machine context
	UpdateContext struct {
		Config      *config.Config
		EventSender *events.EventSender
		Client      *client.GatewayClient

		FromTarget   target.Target
		ToTarget     target.Target
		UpdateRunner update.Runner

		CurrentState ActionName
	}
)

func (u *UpdateContext) SendEvent(event events.EventTypeValue, success ...bool) {
	if err := u.EventSender.EnqueueEvent(event, u.UpdateRunner.Status().ID, u.ToTarget, success...); err != nil {
		slog.Error("failed to send event", "event", event, "err", err)
	}
}
