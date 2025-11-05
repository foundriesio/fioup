// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/pkg/client"
	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/status"
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
		CurrentStatus  *status.CurrentStatus
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
		Targets     target.Targets

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
	var opts []events.EnqueueEventOption
	if len(success) > 0 {
		opts = append(opts, events.WithEventStatus(success[0]))
	}
	opts = append(opts, events.WithEventDetails(u.getEventDetails(event)))
	if err := u.EventSender.EnqueueEvent(event, u.UpdateRunner.Status().ID, u.ToTarget, opts...); err != nil {
		slog.Error("failed to send event", "event", event, "err", err)
	}
}

func (u *UpdateContext) getEventDetails(eventType events.EventTypeValue) string {
	type (
		updateInfo struct {
			Type     UpdateType `json:"type"`
			Mode     UpdateMode `json:"mode"`
			State    string     `json:"state"`
			Progress int        `json:"progress"`
		}
		target struct {
			ID   string   `json:"id"`
			Apps []string `json:"apps"`
		}
		current struct {
			ID   string   `json:"id"`
			Apps []string `json:"apps"`
		}
		updateDetails struct {
			Update  updateInfo `json:"update"`
			Target  target     `json:"target"`
			Current current    `json:"current"`
		}
	)

	switch eventType {
	case events.DownloadStarted:
		updateStatus := u.UpdateRunner.Status()
		updateDetails := updateDetails{
			Update: updateInfo{
				Type:     u.Type,
				Mode:     u.Mode,
				State:    updateStatus.State.String(),
				Progress: updateStatus.Progress,
			},
			Target: target{
				ID:   u.ToTarget.ID,
				Apps: u.ToTarget.AppURIs(),
			},
			Current: current{
				ID:   u.FromTarget.ID,
				Apps: u.FromTarget.AppURIs(),
			},
		}
		data, _ := json.MarshalIndent(updateDetails, "", "  ")
		return string(data)
	}
	return ""
}
