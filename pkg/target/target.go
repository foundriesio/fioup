// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package target

import (
	"encoding/json"
	"log/slog"
	"os"
	"slices"
	"strconv"
)

type (
	App struct {
		Name string `json:"name"`
		URI  string `json:"uri"`
	}
	Apps   []App
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
		Version int                 `json:"version"`
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

func (t *Target) Equals(other *Target) bool {
	if t.ID != other.ID || t.Version != other.Version || len(t.Apps) != len(other.Apps) {
		return false
	}
	for _, app := range t.Apps {
		if !slices.ContainsFunc(other.Apps, func(a App) bool { return a.Name == app.Name && a.URI == app.URI }) {
			return false
		}
	}
	return true
}

func (t *Target) IsUnknown() bool {
	return t.ID == UnknownTarget.ID && t.Version == UnknownTarget.Version
}

func (t *Target) NoApps() bool {
	return len(t.Apps) == 0
}

// ShortlistApps filters the target's apps to only include those whose name is in the provided shortlist.
// If the shortlist is nil then it does not filter the apps.
// If the shortlist is empty then it clears the target's apps.
// This should be used for shortlisting based on "compose_apps" value in the .toml config.
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

// ShortlistAppsByURI filters the target's apps to only include those whose URI is in the provided shortlist.
// If the shortlist is nil OR empty, it clears the target's apps.
// This should be used solely for shortlisting based on an update record's "uri" field,
// which interprets both nil and empty as update to the state with no apps.
func (t *Target) ShortlistAppsByURI(shortlist []string) {
	if shortlist == nil {
		t.Apps = nil
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

func (a Apps) Names() (res []string) {
	for _, app := range a {
		res = append(res, app.Name)
	}
	return
}

func (a Apps) URIs() (res []string) {
	res = []string{}
	for _, app := range a {
		res = append(res, app.URI)
	}
	return
}

func (t *Target) AppURIs() (res []string) {
	for _, app := range t.Apps {
		res = append(res, app.URI)
	}
	return
}

func (t *Target) Diff(other *Target) (added, removed, same, from, to Apps) {
	appMap := make(map[string]App)
	for _, app := range t.Apps {
		appMap[app.Name] = app
	}
	for _, app := range other.Apps {
		if _, exists := appMap[app.Name]; !exists {
			added = append(added, app)
		} else {
			if app.URI != appMap[app.Name].URI {
				from = append(from, appMap[app.Name])
				to = append(to, app)
			} else {
				same = append(same, app)
			}
			delete(appMap, app.Name)
		}
	}
	for _, app := range appMap {
		removed = append(removed, app)
	}
	return
}

func (t Targets) GetLatestTarget() Target {
	versionLimitStr := os.Getenv("FIOUP_VERSION_UPPER_LIMIT")
	versionLimit := 0
	if versionLimitStr != "" {
		var err error
		versionLimit, err = strconv.Atoi(versionLimitStr)
		if err != nil {
			slog.Debug("Invalid value for FIOUP_VERSION_UPPER_LIMIT. Ignoring it", "value", versionLimitStr)
		}
		slog.Info("Enforcing upper version limit", "version_limit", versionLimit)
	}

	latest := UnknownTarget
	for _, target := range t {
		if target.Version > latest.Version && (versionLimit <= 0 || target.Version <= versionLimit) {
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
