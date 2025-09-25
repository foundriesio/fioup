// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package target

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/fiotuf/tuf"
	"github.com/foundriesio/fioup/pkg/fioup/client"
	"github.com/foundriesio/fioup/pkg/fioup/config"
)

type (
	tufRepo struct {
		dgClient   *client.GatewayClient
		tufClient  *tuf.FioTuf
		targets    Targets
		hardwareID string
	}
)

func NewTufRepo(cfg *config.Config, dgClient *client.GatewayClient, hardwareID string) (Repo, error) {
	tufClient, err := tuf.NewFioTuf(cfg.TomlConfig(), dgClient.HttpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUF HttpClient to talk to TUF repo: %w", err)
	}
	return &tufRepo{
		dgClient:   dgClient,
		tufClient:  tufClient,
		hardwareID: hardwareID,
	}, nil
}

func (r *tufRepo) update() error {
	// We need to figure out the way set headers (r.dgClient.Headers) to r.tufClient, so it adds
	// headers we need to the requests it makes to DG
	if err := r.tufClient.RefreshTuf(""); err != nil {
		return fmt.Errorf("failed to update TUF metadata: %w", err)
	}
	return r.loadTargets()
}

func (r *tufRepo) LoadTargets(update bool) (Targets, error) {
	if update {
		if err := r.update(); err != nil {
			return nil, err
		}
	} else {
		if err := r.loadTargets(); err != nil {
			return nil, err
		}
	}
	return r.targets, nil
}

func (r *tufRepo) loadTargets() error {
	r.targets = nil
	for id, targetValue := range r.tufClient.GetTargets() {
		var targetDetails Custom
		var b []byte
		b, _ = targetValue.Custom.MarshalJSON()
		err := json.Unmarshal(b, &targetDetails)
		if err != nil {
			slog.Debug("invalid value of target custom field is found", "target custom", targetValue)
			continue
		}

		if len(targetDetails.HardwareID) == 0 {
			slog.Debug("target with no hardware ID is found", "target custom", targetValue)
			continue
		}
		var match bool
		for _, hwID := range targetDetails.HardwareID {
			if hwID == r.hardwareID {
				match = true
				break
			}
		}
		if !match {
			continue
		}
		version, err := strconv.Atoi(targetDetails.Version)
		if err != nil {
			slog.Debug("invalid value of target version is found", "target custom", targetValue)
			continue
		}

		var apps []App
		for _, appField := range targetDetails.Apps {
			appRef, err := compose.ParseAppRef(appField.URI)
			if err != nil {
				slog.Debug("target with invalid app URI is found", "target custom", targetDetails)
				continue
			}
			apps = append(apps, App{
				Name: appRef.Name,
				URI:  appField.URI,
			})
		}

		r.targets = append(r.targets, Target{
			ID:      id,
			Version: version,
			Apps:    apps,
		})
	}
	return nil
}
