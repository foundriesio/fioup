// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package target

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/fioup/pkg/client"
)

type (
	plainRepo struct {
		dgClient        *client.GatewayClient
		targetsFilepath string
		targets         []Target
		hardwareID      string
		version         int
	}
)

const (
	TargetsResourcePath = "/repo/targets.json"
)

func NewPlainRepo(dgClient *client.GatewayClient, targetsFilepath string, hardwareID string) (Repo, error) {
	return &plainRepo{
		dgClient:        dgClient,
		targetsFilepath: targetsFilepath,
		hardwareID:      hardwareID,
	}, nil
}

func (r *plainRepo) update() error {
	res, err := r.dgClient.Get(TargetsResourcePath)
	if err != nil {
		return fmt.Errorf("failed to get targets from Device Gateway: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code HTTP_%d from %s: %s", res.StatusCode, TargetsResourcePath, res.String())
	}
	var targetsFile File
	if err := res.Json(&targetsFile); err != nil {
		return fmt.Errorf("failed to unmarshal 'targets.json' received from Device Gateway: %w", err)
	}

	if err := os.WriteFile(r.targetsFilepath, res.Body, 0644); err != nil {
		return fmt.Errorf("failed to write obtained targets to file: %w", err)
	}
	return r.loadTargets(res.Body)
}

func (r *plainRepo) LoadTargets(update bool) (Targets, int, error) {
	if update {
		if err := r.update(); err != nil {
			return nil, -1, err
		}
	} else {
		if err := r.readTargets(); err != nil {
			return nil, -1, err
		}
	}
	return r.targets, r.version, nil
}

func (r *plainRepo) readTargets() error {
	b, err := os.ReadFile(r.targetsFilepath)
	if err != nil {
		return fmt.Errorf("failed to read targets from file: %w", err)
	}
	return r.loadTargets(b)
}

func (r *plainRepo) loadTargets(targetsData []byte) error {
	var targetsFile File
	if err := json.Unmarshal(targetsData, &targetsFile); err != nil {
		return fmt.Errorf("failed to unmarshal 'targets.json' read from file: %w", err)
	}
	r.targets = nil
	for targetName, targetValue := range targetsFile.Signed.Targets {
		version, err := strconv.Atoi(targetValue.Custom.Version)
		if err != nil {
			slog.Debug("invalid value of target version is found", "target custom", targetValue.Custom)
			continue
		}
		if len(targetValue.Custom.HardwareID) == 0 {
			slog.Debug("target with no hardware ID is found", "target custom", targetValue.Custom)
			continue
		}
		var match bool
		for _, hwID := range targetValue.Custom.HardwareID {
			if hwID == r.hardwareID {
				match = true
				break
			}
		}
		if !match {
			continue
		}
		var apps []App
		for _, appField := range targetValue.Custom.Apps {
			appRef, err := compose.ParseAppRef(appField.URI)
			if err != nil {
				slog.Debug("target with invalid app URI is found", "target custom", targetValue.Custom)
				continue
			}
			apps = append(apps, App{
				Name: appRef.Name,
				URI:  appField.URI,
			})
		}
		r.targets = append(r.targets, Target{
			ID:      targetName,
			Version: version,
			Apps:    apps,
		})
	}
	r.version = targetsFile.Signed.Version
	return nil
}
