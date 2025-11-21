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
	opts = append(opts, events.WithEventDetails(u.getEventDetails(event, eventError)))
	if err := u.EventSender.EnqueueEvent(event, u.UpdateRunner.Status().ID, u.ToTarget, opts...); err != nil {
		slog.Error("failed to send event", "event", event, "err", err)
	}
}

func (u *UpdateContext) getEventDetails(eventType events.EventTypeValue, eventError error) string {
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
		fetchSize struct {
			Bytes int64 `json:"bytes"`
			Blobs int   `json:"blobs"`
		}
		downloadCompleteDetails struct {
			Fetched fetchSize `json:"fetched"`
			Error   string    `json:"error,omitempty"`
		}
	)
	var detailsByte []byte

	updateStatus := u.UpdateRunner.Status()
	switch eventType {
	case events.DownloadStarted:
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
		detailsByte, _ = json.MarshalIndent(updateDetails, "", "  ")
	case events.DownloadCompleted:
		downloadCompleteDetails := downloadCompleteDetails{
			Fetched: fetchSize{
				Bytes: updateStatus.FetchedBytes,
				Blobs: updateStatus.FetchedBlobs,
			},
		}
		if eventError != nil {
			downloadCompleteDetails.Error = eventError.Error()
		}
		detailsByte, _ = json.MarshalIndent(downloadCompleteDetails, "", "  ")
	case events.InstallationCompleted:
		var installCompleteDetails struct {
			AppStatuses []status.AppStatus `json:"app_statuses"`
			Error       string             `json:"error,omitempty"`
		}
		for _, appStatus := range u.CurrentStatus.AppStatuses {
			installCompleteDetails.AppStatuses = append(installCompleteDetails.AppStatuses, appStatus)
		}
		if eventError != nil {
			installCompleteDetails.Error = eventError.Error()
		}
		detailsByte, _ = json.MarshalIndent(installCompleteDetails, "", "  ")
	}
	return string(detailsByte)
}
