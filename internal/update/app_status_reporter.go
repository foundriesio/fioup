// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package update

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/fioconfig/sotatoml"
	"github.com/foundriesio/fioup/pkg/fioup/client"
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

	AppsState struct {
		Ostree     string              `json:"ostree"`
		DeviceTime string              `json:"deviceTime"`
		Apps       map[string]AppState `json:"apps"`
	}
)

func ReportAppsStates(config *sotatoml.AppConfig, client *client.GatewayClient, updateContext *UpdateContext) error {
	// Get status of all apps found in the local app storage
	status, err := compose.CheckAppsStatus(updateContext.Context, updateContext.ComposeConfig, nil)
	if err != nil {
		return err
	}
	appStates := AppsState{
		Ostree:     "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", // hash of empty string
		DeviceTime: time.Now().UTC().Format(time.RFC3339),
		Apps:       make(map[string]AppState),
	}
	for _, app := range status.Apps {
		appRunningStatus, ok := status.AppsRunningStatus[app.Ref().Digest]
		if !ok {
			continue
		}
		var appServices []AppServiceState
		state := "healthy"
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
			if srv.Health == "unhealthy" {
				state = "unhealthy"
			}
		}
		appStates.Apps[app.Name()] = AppState{
			Uri:      app.Ref().String(),
			State:    state,
			Services: appServices,
		}
	}
	data, err := json.Marshal(appStates)
	if err != nil {
		return err
	}
	res, err := client.Post("/apps-states", data)
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode > 204 {
		return fmt.Errorf("failed to post status of apps: HTTP_%d - %s", res.StatusCode, res)
	}
	return nil
}
