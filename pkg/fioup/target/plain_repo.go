// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package target

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/fioup/pkg/fioup/client"
)

type (
	plainRepo struct {
		dgClient        *client.GatewayClient
		targetsFilepath string
		targets         []Target
	}
)

const (
	TargetsResourcePath = "/repo/targets.json"
)

func NewPlainRepo(dgClient *client.GatewayClient, targetsFilepath string) (Repo, error) {
	return &plainRepo{
		dgClient:        dgClient,
		targetsFilepath: targetsFilepath,
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

func (r *plainRepo) LoadTargets(update bool) (Targets, error) {
	if update {
		if err := r.update(); err != nil {
			return nil, err
		}
	} else {
		if err := r.readTargets(); err != nil {
			return nil, err
		}
	}
	return r.targets, nil
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
	//hardwareID := r.cfg.GetHardwareID()
	for targetName, targetValue := range targetsFile.Signed.Targets {
		version, err := strconv.Atoi(targetValue.Custom.Version)
		if err != nil {
			// TODO: add debug level log about failing to get target version
			continue
		}
		if len(targetValue.Custom.HardwareID) == 0 {
			// TODO: add debug level log about detecting target without hardware ID
			continue
		}
		// TODO: consider filtering out by arch
		//var match bool
		//for _, hwID := range targetValue.Custom.HardwareID {
		//
		//	if hwID == hardwareID {
		//		match = true
		//		break
		//	}
		//}
		//if !match {
		//	continue
		//}
		var apps []App
		for _, appField := range targetValue.Custom.Apps {
			appRef, err := compose.ParseAppRef(appField.URI)
			if err != nil {
				// TODO: add debug level log about invalid app URI
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
	return nil
}
