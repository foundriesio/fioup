// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package target

import (
	"encoding/json"
	"fmt"
	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/fioconfig/transport"
	"github.com/foundriesio/fioup/internal/update"
	"github.com/foundriesio/fioup/pkg/fioup/config"
	tuf "github.com/theupdateframework/go-tuf/v2/metadata"
	"net/http"
	"strconv"
)

type (
	Provider interface {
		UpdateTargets(cfg *config.Config) error
		GetTargetByName(name string) (Target, error)
		GetTargetByVersion(version int) (Target, error)
		GetLatestTarget() (Target, error)
	}
	Target interface {
		Name() string
		Version() int
		Apps() []string
	}
	targetProvider struct {
		client  *http.Client
		targets []Target
	}
	target struct {
		*tuf.TargetFiles
		version int
		apps    []string
	}
	unknownTarget struct {
		name    string
		version int
		apps    []string
	}
)

func NewTargetProvider(cfg *config.Config) (Provider, error) {
	client, err := transport.CreateClient(cfg.TomlConfig())
	if err != nil {
		return nil, err
	}
	targets, err := update.GetTargets(cfg.TomlConfig(),
		"",
		client,
		false,
		"",
		cfg.GetEnabledApps())
	if err != nil {
		return nil, err
	}
	p := &targetProvider{
		client: client,
	}
	for _, t := range targets {
		target, err := NewTarget(t, cfg.GetEnabledApps())
		if err != nil {
			return nil, err
		}
		p.targets = append(p.targets, target)
	}
	return p, nil
}

func (p *targetProvider) UpdateTargets(cfg *config.Config) error {
	// TODO: implement fetching targets.json from DG, the other types of fetching targets should
	// be implemented in other Provider implementations:
	// 1. Local provider - read from local file
	// 2. TUF remote provider
	// 3. TUF local provider
	targets, err := update.GetTargets(cfg.TomlConfig(),
		"",
		p.client,
		true,
		"",
		cfg.GetEnabledApps())
	if err != nil {
		return err
	}
	for _, t := range targets {
		target, err := NewTarget(t, cfg.GetEnabledApps())
		if err != nil {
			return err
		}
		p.targets = append(p.targets, target)
	}
	return err
}

func (p *targetProvider) GetTargetByName(name string) (Target, error) {
	for _, t := range p.targets {
		if t.Name() == name {
			return t, nil
		}
	}
	return nil, fmt.Errorf("target %s not found", name)
}

func (p *targetProvider) GetTargetByVersion(version int) (Target, error) {
	for _, t := range p.targets {
		if t.Version() == version {
			return t, nil
		}
	}
	return nil, fmt.Errorf("target %d not found", version)
}

func (p *targetProvider) GetLatestTarget() (Target, error) {
	var latestTarget Target
	latestVersion := -1
	for _, t := range p.targets {
		if t.Version() > latestVersion {
			latestVersion = t.Version()
			latestTarget = t
		}
	}
	if latestVersion != -1 && latestTarget != nil {
		return latestTarget, nil
	}
	return nil, fmt.Errorf("latest target not found")
}

func NewUnknownTarget() Target {
	return &unknownTarget{
		name:    "Unknown",
		version: -1,
		apps:    []string{},
	}
}

func (t *unknownTarget) Name() string {
	return t.name
}

func (t *unknownTarget) Version() int {
	return t.version
}

func (t *unknownTarget) Apps() []string {
	return t.apps
}

func NewTarget(tufTarget *tuf.TargetFiles, appShortlist []string) (Target, error) {
	b, err := tufTarget.Custom.MarshalJSON()
	if err != nil {
		return nil, err
	}
	t := target{
		TargetFiles: tufTarget,
	}
	type targetCustom struct {
		Version string `json:"version"`
	}
	var custom targetCustom
	err = json.Unmarshal(b, &custom)
	if err != nil {
		return nil, err
	}
	t.version, err = strconv.Atoi(custom.Version)
	if err != nil {
		return nil, err
	}
	appURIs, err := update.GetAppsUris(t.TargetFiles)
	if err != nil {
		return nil, err
	}
	if len(appShortlist) == 0 {
		t.apps = appURIs
		return &t, nil
	}
	for _, app := range appURIs {
		appRef, err := compose.ParseAppRef(app)
		if err != nil {
			return nil, err
		}
		for _, shortlistApp := range appShortlist {
			if appRef.Name == shortlistApp {
				t.apps = append(t.apps, app)
			}
		}
	}
	return &t, nil
}

func (t *target) Name() string {
	return t.Path
}

func (t *target) Version() int {
	return t.version
}

func (t *target) Apps() []string {
	return t.apps
}
