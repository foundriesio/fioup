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

	UpdateSize struct {
		Bytes int64 `json:"bytes"`
		Blobs int   `json:"blobs"`
	}

	UpdateInfo struct {
		TotalStates     int
		CurrentStateNum int
		CurrentState    ActionName
		FromTarget      target.Target
		ToTarget        target.Target
		Mode            UpdateMode
		Type            UpdateType
		Size            UpdateSize
		AppDiff         struct {
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

	downloadDetails struct {
		Fetched  UpdateSize `json:"fetched"`
		Progress int        `json:"progress_percent"`
	}
	downloadStartedDetails struct {
		ToBeFetched     UpdateSize `json:"total_to_be_fetched"`
		downloadDetails `json:",inline"`
	}
	downloadCompletedDetails struct {
		Error           string `json:"error,omitempty"`
		downloadDetails `json:",inline"`
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

func (u *UpdateContext) getDownloadDetails() downloadDetails {
	s := u.UpdateRunner.Status()
	details := downloadDetails{
		Fetched: UpdateSize{
			Bytes: s.FetchedBytes,
			Blobs: s.FetchedBlobs,
		},
	}
	if u.Size.Bytes == 0 {
		// Nothing to fetch, consider as 100% fetched
		details.Progress = 100
	} else if s.State == update.StateFetching || s.State == update.StateFetched {
		details.Progress = int((float64(s.FetchedBytes) / float64(u.Size.Bytes)) * 100)
	}
	return details
}

func (u *UpdateContext) getDownloadStartedDetails() interface{} {
	return &downloadStartedDetails{
		ToBeFetched:     u.Size,
		downloadDetails: u.getDownloadDetails(),
	}
}

func (u *UpdateContext) getDownloadCompletedDetails(eventErr error) interface{} {
	details := &downloadCompletedDetails{
		downloadDetails: u.getDownloadDetails(),
	}
	if eventErr != nil {
		details.Error = eventErr.Error()
	}
	return details
}

func (u *UpdateContext) getInstallationStartedDetails() interface{} {
	type installationActions struct {
		ToBeStopped   []string `json:"stop"`
		ToBeInstalled []string `json:"install"`
		ToBePruned    []string `json:"prune"`
	}
	return &struct {
		Actions installationActions `json:"actions"`
	}{
		Actions: installationActions{
			ToBeStopped:   u.FromTarget.AppURIs(),
			ToBeInstalled: u.ToTarget.AppURIs(),
			ToBePruned:    append(u.AppDiff.Remove.URIs(), u.AppDiff.Update.URIs()...),
		},
	}
}

func (u *UpdateContext) getInstallationCompletedDetails(eventErr error) interface{} {
	type installationCompletedDetails struct {
		Error       string             `json:"error,omitempty"`
		AppStatuses []status.AppStatus `json:"current_status,omitempty"`
	}
	var details installationCompletedDetails
	if u.CurrentStatus != nil {
		details.AppStatuses = u.CurrentStatus.AppStatusList()
	}
	if eventErr != nil {
		details.Error = eventErr.Error()
	}
	return &details
}
