// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"time"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/pkg/errors"
)

type (
	AppServiceState struct {
		Name     string `json:"name"`
		Hash     string `json:"hash"`
		State    string `json:"state"`
		Status   string `json:"status"`
		Health   string `json:"health,omitempty"`
		ImageUri string `json:"image"`
		Logs     string `json:"logs,omitempty"`
	}

	AppState struct {
		Services []AppServiceState `json:"services"`
		State    string            `json:"state"`
		Uri      string            `json:"uri"`
	}

	AppStates struct {
		Ostree     string              `json:"ostree"`
		DeviceTime string              `json:"deviceTime"`
		Apps       map[string]AppState `json:"apps"`
	}
)

func (c *GatewayClient) initAppStateReporter() {
	if b, err := os.ReadFile(c.lastAppStatesFile); err == nil {
		if err := json.Unmarshal(b, &c.lastAppStates); err != nil {
			slog.Debug("failed to unmarshal last app states from file", "error", err)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		slog.Debug("failed to read last app states from file", "error", err)
	}
}

func (c *GatewayClient) ReportAppStates(ctx context.Context, cfg *compose.Config) error {
	currentAppStates, err := getAppStates(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to get app states: %w", err)
	}
	if areAppStatesEqual(currentAppStates, c.lastAppStates) {
		// No change in app states
		slog.Debug("no change in app states; skipping reporting to device gateway")
		return nil
	}
	statusToReport := AppStates{
		Ostree:     "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", // hash of empty string
		DeviceTime: time.Now().UTC().Format(time.RFC3339),
		Apps:       currentAppStates,
	}
	b, err := json.Marshal(statusToReport)
	if err != nil {
		return fmt.Errorf("failed to marshal status of apps: %w", err)
	}
	res, err := c.Post("/apps-states", b)
	if err != nil {
		return fmt.Errorf("failed to post status of apps: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode > 204 {
		return fmt.Errorf("failed to post status of apps: HTTP_%d - %s", res.StatusCode, res)
	}
	if b, err := json.Marshal(currentAppStates); err == nil {
		if err := os.WriteFile(c.lastAppStatesFile, b, 0o744); err != nil {
			slog.Debug("failed to write last app states to file", "error", err)
		}
	} else {
		slog.Debug("failed to marshal last app states", "error", err)
	}
	c.lastAppStates = currentAppStates
	return nil
}

func getAppStates(ctx context.Context, cfg *compose.Config) (appStates map[string]AppState, err error) {
	// Get status of all apps found in the local app storage
	status, err := compose.CheckAppsStatus(ctx, cfg, nil)
	if err != nil {
		return
	}
	appStates = make(map[string]AppState)
	for _, app := range status.Apps {
		appRunningStatus, ok := status.AppsRunningStatus[app.Ref().Digest]
		if !ok {
			continue
		}
		var appServices []AppServiceState
		var appState string
		if len(appRunningStatus.Services) > 0 {
			appState = "healthy"
		} else {
			appState = "unhealthy"
		}
		for _, srv := range appRunningStatus.Services {
			appServices = append(appServices, AppServiceState{
				Name:     srv.Name,
				Hash:     srv.Hash,
				State:    srv.State,
				Status:   srv.Status,
				Health:   srv.Health,
				ImageUri: srv.Image,
				Logs:     "", // TODO: implement logs retrieval
			})
			if srv.State == "not created" || srv.Health == "unhealthy" {
				appState = "unhealthy"
			}
		}
		appStates[app.Name()] = AppState{
			Uri:      app.Ref().String(),
			State:    appState,
			Services: appServices,
		}
	}
	return
}

func areAppStatesEqual(a, b map[string]AppState) bool {
	// Compare two slices of AppState for equality, ignoring the order of services
	if len(a) != len(b) {
		return false
	}
	for uri, appA := range a {
		appB, ok := b[uri]
		if !ok {
			// App with this URI not found in b
			return false
		}
		if appA.State != appB.State || len(appA.Services) != len(appB.Services) {
			return false
		}
	}
	return true
}
