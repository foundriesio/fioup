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

func (u *UpdateContext) SendEvent(event events.EventTypeValue, eventErr ...error) {
	var opts []events.EnqueueEventOption
	var eventError error
	if len(eventErr) > 0 {
		eventStatus := eventErr[0] == nil
		opts = append(opts, events.WithEventStatus(eventStatus))
		eventError = eventErr[0]
	}
	if details := u.getUpdateDetails(event, eventError); details != "" {
		opts = append(opts, events.WithEventDetails(details))
	}
	if err := u.EventSender.EnqueueEvent(event, u.UpdateRunner.Status().ID, u.ToTarget, opts...); err != nil {
		slog.Error("failed to send event", "event", event, "err", err)
	}
}

func (u *UpdateContext) getUpdateDetails(eventType events.EventTypeValue, eventErr error) string {
	var detailsString string
	var details interface{}
	switch eventType {
	case events.DownloadStarted:
		details = u.getDownloadStartedDetails()
	case events.DownloadCompleted:
		details = u.getDownloadCompletedDetails(eventErr)
	case events.InstallationStarted:
		details = u.getInstallationStartedDetails()
	case events.InstallationCompleted:
		details = u.getInstallationCompletedDetails(eventErr)
	}
	if details == nil {
		return ""
	}
	if detailBytes, err := json.MarshalIndent(details, "", "  "); err == nil {
		detailsString = string(detailBytes)
	} else {
		slog.Error("failed to marshal event details; no any details will be added to the update event context",
			"event", eventType, "err", err)
	}
	return detailsString
}

func (u *UpdateContext) getDownloadStartedDetails() interface{} {
	return nil
}

func (u *UpdateContext) getDownloadCompletedDetails(eventErr error) interface{} {
	return nil
}

func (u *UpdateContext) getInstallationStartedDetails() interface{} {
	return nil
}

func (u *UpdateContext) getInstallationCompletedDetails(eventErr error) interface{} {
	return nil
}
