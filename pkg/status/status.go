// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package status

import (
	"context"
	"fmt"
	"time"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
)

type (
	AppStatus struct {
		URI       string `json:"uri"`
		Name      string `json:"name"`
		Fetched   bool   `json:"fetched"`
		Installed bool   `json:"installed"`
		Running   bool   `json:"running"`
	}

	CurrentStatus struct {
		UpdateID    string      `json:"update_id"`
		TargetID    string      `json:"target_id"`
		AppStatuses []AppStatus `json:"apps"`
		CompletedAt time.Time   `json:"completed_at"`
	}

	BlobStats struct {
		Bytes    int64 `json:"bytes"`
		NumBlobs int   `json:"num_blobs"`
	}

	UpdateStatus struct {
		ID          string       `json:"id"`
		TargetID    string       `json:"target_id"`
		State       update.State `json:"state"`
		StartTime   time.Time    `json:"start_time"`
		UpdatedAt   time.Time    `json:"updated_at"`
		Apps        []string     `json:"apps"`
		Size        BlobStats    `json:"size"`
		FetchedSize BlobStats    `json:"fetched_size"`
		Progress    int          `json:"progress"`
	}
)

func GetCurrentStatus(ctx context.Context, cfg *compose.Config) (*CurrentStatus, error) {
	currentStatus := CurrentStatus{
		AppStatuses: []AppStatus{},
	}
	var appURIs []string

	if lastUpdate, err := update.GetLastSuccessfulUpdate(cfg); err == nil {
		appURIs = lastUpdate.URIs
		currentStatus.UpdateID = lastUpdate.ID
		currentStatus.TargetID = lastUpdate.ClientRef
		currentStatus.CompletedAt = lastUpdate.UpdateTime
	} else {
		// No successful last update exists, falls back to the status of apps detected locally;
		// if no apps are specified to `CheckAppsStatus` then it will check all apps found locally
		currentStatus.TargetID = "unknown"
	}
	s, err := compose.CheckAppsStatus(ctx, cfg, appURIs, compose.WithQuickCheckFetch(true))
	if err != nil {
		return nil, fmt.Errorf("failed to check apps' status: %w", err)
	}

	areFetched := s.AreFetched()
	areInstalled := s.AreInstalled()
	areRunning := s.AreRunning()

	for _, app := range s.Apps {
		currentStatus.AppStatuses = append(currentStatus.AppStatuses, AppStatus{
			URI:  app.Ref().String(),
			Name: app.Name(),
			// TODO: per-app status check instead of global, add wrapper functions over `compose.AppsStatus` for that
			Fetched:   areFetched,
			Installed: areInstalled,
			Running:   areRunning,
		})
	}
	return &currentStatus, nil
}

func GetUpdateStatus(cfg *compose.Config) (*UpdateStatus, error) {
	s, err := update.GetLastUpdate(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get last update: %w", err)
	}
	return &UpdateStatus{
		ID:        s.ID,
		TargetID:  s.ClientRef,
		State:     s.State,
		StartTime: s.CreationTime,
		UpdatedAt: s.UpdateTime,
		Apps:      s.URIs,
		Size: BlobStats{
			Bytes:    s.TotalBlobsBytes,
			NumBlobs: len(s.Blobs),
		},
		FetchedSize: BlobStats{
			Bytes:    s.FetchedBytes,
			NumBlobs: s.FetchedBlobs,
		},
		Progress: s.Progress,
	}, nil
}
