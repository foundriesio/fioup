// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"log/slog"
	"time"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/pkg/client"
	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/target"
)

type (
	// UpdateType defines the update type, update to another version or sync the current version
	UpdateType string
	// UpdateMode defines the update mode, new or resume
	UpdateMode string
	// ActionName Name of the state action
	ActionName string
	// ActionState interface for all states
	ActionState interface {
		Name() ActionName
		Execute(ctx context.Context, updateCtx *UpdateContext) error
	}

	UpdateInfo struct {
		TotalStates     int
		CurrentStateNum int
		CurrentState    ActionName
		FromTarget      target.Target
		ToTarget        target.Target
		Mode            UpdateMode
		Type            UpdateType
		Size            struct {
			Bytes int64
			Blobs int
		}
		AppDiff struct {
			Remove target.Apps
			Add    target.Apps
			Sync   target.Apps
			Update target.Apps
		}
		InitializedAt  time.Time
		FetchedAt      time.Time
		AlreadyFetched bool
	}

	// UpdateContext holds the state machine context
	UpdateContext struct {
		UpdateInfo

		Config      *config.Config
		EventSender *events.EventSender
		Client      *client.GatewayClient

		UpdateRunner update.Runner
	}
)

var (
	UpdateModeNewUpdate UpdateMode = "new"
	UpdateModeResume    UpdateMode = "resume"
	UpdateTypeUpdate    UpdateType = "update"
	UpdateTypeSync      UpdateType = "sync"
)

func (u *UpdateContext) SendEvent(event events.EventTypeValue, success ...bool) {
	if err := u.EventSender.EnqueueEvent(event, u.UpdateRunner.Status().ID, u.ToTarget, success...); err != nil {
		slog.Error("failed to send event", "event", event, "err", err)
	}
}
