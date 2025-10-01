// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package target

import (
	"encoding/json"
	"slices"
)

type (
	App struct {
		Name string `json:"name"`
		URI  string `json:"uri"`
	}
	Target struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
		Apps    []App  `json:"apps"`
	}
	Targets []Target

	File struct {
		Signatures *json.RawMessage `json:"signatures"`
		Signed     Signed           `json:"signed"`
	}
	Signed struct {
		Targets map[string]Metadata `json:"targets"`
	}
	Metadata struct {
		Custom Custom `json:"custom"`
	}
	Custom struct {
		Version string `json:"version"`
		Apps    map[string]struct {
			URI string `json:"uri"`
		} `json:"docker_compose_apps"`
		Arch       string   `json:"arch"`
		HardwareID []string `json:"hardwareIds"`
	}
)

var (
	UnknownTarget = Target{
		ID:      "unknown",
		Version: -1,
	}
)

func (t *Target) IsUnknown() bool {
	return t.ID == UnknownTarget.ID && t.Version == UnknownTarget.Version
}

func (t *Target) NoApps() bool {
	return len(t.Apps) == 0
}

func (t *Target) ShortlistApps(shortlist []string) {
	if shortlist == nil {
		return
	}
	var shortlistedApps []App
	for _, app := range t.Apps {
		var found bool
		for _, a := range shortlist {
			if app.Name == a {
				found = true
				break
			}
		}
		if found {
			shortlistedApps = append(shortlistedApps, app)
		}
	}
	t.Apps = shortlistedApps
}

func (t *Target) ShortlistAppsByURI(shortlist []string) {
	if shortlist == nil {
		return
	}
	var shortlistedApps []App
	for _, app := range t.Apps {
		var found bool
		for _, a := range shortlist {
			if app.URI == a {
				found = true
				break
			}
		}
		if found {
			shortlistedApps = append(shortlistedApps, app)
		}
	}
	t.Apps = shortlistedApps
}

func (t *Target) AppNames() (res []string) {
	for _, app := range t.Apps {
		res = append(res, app.Name)
	}
	return
}

func (t *Target) AppURIs() (res []string) {
	for _, app := range t.Apps {
		res = append(res, app.URI)
	}
	return
}

func (t Targets) GetLatestTarget() Target {
	latest := UnknownTarget
	for _, target := range t {
		if target.Version > latest.Version {
			latest = target
		}
	}
	return latest
}

func (t Targets) GetTargetByVersion(version int) Target {
	for _, t := range t {
		if t.Version == version {
			return t
		}
	}
	return UnknownTarget
}

func (t Targets) GetTargetByID(ID string) Target {
	for _, t := range t {
		if t.ID == ID {
			return t
		}
	}
	return UnknownTarget
}

func (t Targets) GetSortedList() Targets {
	cp := slices.Clone(t)

	slices.SortFunc(cp, func(a, b Target) int {
		return a.Version - b.Version
	})
	return cp
}
