// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/pkg/status"
	"github.com/foundriesio/fioup/pkg/target"
	"github.com/pkg/errors"
)

type (
	Check struct {
		UpdateTargets  bool
		AllowNewUpdate bool
		Force          bool
		Action         string
		AllowedStates  []update.State
		ToVersion      int
		SyncCurrent    bool
		MaxAttempts    int
		RequireLatest  bool
		EnableTUF      bool
	}
)

var (
	ErrCheckNoUpdate           = errors.New("selected target is already running")
	ErrNewerVersionIsAvailable = errors.New("can't resume current update, there is a newer version available")
	ErrInvalidOngoingUpdate    = errors.New("invalid ongoing update")
)

func (s *Check) Name() ActionName { return "Checking" }
func (s *Check) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	var err error

	// Check if there is an ongoing update and set action type accordingly and fail if given action is not allowed
	updateCtx.UpdateRunner, err = update.GetCurrentUpdate(updateCtx.Config.ComposeConfig())
	if errors.Is(err, update.ErrUpdateNotFound) {
		if !s.AllowNewUpdate {
			return fmt.Errorf("no ongoing update to %s found;"+
				" please run %q or %q first", s.Action, "fioup update", "fioup fetch")
		}
		updateCtx.Mode = UpdateModeNewUpdate
		err = nil
	} else {
		updateCtx.Mode = UpdateModeResume
	}
	if err != nil {
		return fmt.Errorf("failed to get info about current update: %w", err)
	}

	// Check if action is allowed at the current state of the update
	if len(s.AllowedStates) > 0 {
		currentState := updateCtx.UpdateRunner.Status().State
		if !currentState.IsOneOf(s.AllowedStates...) {
			return fmt.Errorf("cannot %s current update if it is in state %q", s.Action, currentState)
		}
	}

	var targetRepo target.Repo
	if s.EnableTUF {
		targetRepo, err = target.NewTufRepo(updateCtx.Config, updateCtx.Client, updateCtx.Config.GetHardwareID())
	} else {
		targetRepo, err = target.NewPlainRepo(updateCtx.Client, updateCtx.Config.GetTargetsFilepath(), updateCtx.Config.GetHardwareID())
	}
	if err != nil {
		return err
	}
	targets, err := targetRepo.LoadTargets(s.UpdateTargets)
	if err != nil {
		return err
	}
	updateCtx.Targets = targets

	// Get FromTarget: get last successful update to set FromTarget
	if err := updateCtx.getAndSetCurrentTarget(); err == nil {
		updateCtx.Client.UpdateHeaders(updateCtx.FromTarget.AppNames(), updateCtx.FromTarget.ID)
	} else {
		slog.Debug("failed to determine the current target, consider it as unknown", "error", err)
	}

	if err := updateCtx.selectToTarget(s); err != nil {
		return err
	}

	updateCtx.ToTarget.ShortlistApps(updateCtx.Config.GetEnabledApps())
	if updateCtx.ToTarget.ID == updateCtx.FromTarget.ID {
		updateCtx.Type = UpdateTypeSync
	} else {
		updateCtx.Type = UpdateTypeUpdate
	}
	updateCtx.AppDiff.Add, updateCtx.AppDiff.Remove, updateCtx.AppDiff.Sync,
		updateCtx.AppDiff.Update, updateCtx.AppDiff.UpdateTo = updateCtx.FromTarget.Diff(&updateCtx.ToTarget)
	if updateCtx.UpdateRunner != nil {
		updateCtx.InitializedAt = updateCtx.UpdateRunner.Status().InitTime
	}
	if !s.Force && !updateCtx.isUpdateRequired(ctx) {
		// If this is not a forced new sync update, then check the current status and decide whether the sync is needed.
		// If all current target apps are fetched, installed, and running, then no update is needed - hence ErrCheckNoUpdate is returned.
		slog.Info("Skipping running target", "target_id", updateCtx.ToTarget.ID)
		return ErrCheckNoUpdate
	}
	return nil
}

func (u *UpdateContext) selectToTarget(s *Check) error {
	if u.Mode == UpdateModeResume {
		// Get ToTarget if resuming update
		target, err := u.getOngoingUpdateTarget()
		if err != nil {
			return ErrInvalidOngoingUpdate
		}
		u.ToTarget = *target
		if s.ToVersion != -1 {
			// make sure ToVersion matches the ongoing update, otherwise fail
			if u.ToTarget.Version != s.ToVersion {
				return fmt.Errorf("cannot start or resume update to version %d since there is an ongoing update to version %d",
					s.ToVersion, u.ToTarget.Version)
			}
		} else {
			if s.RequireLatest {
				// When running in daemon mode, do not resume the ongoing update if the latest target has changed
				// Calling code is expected to cancel the ongoing update in this case
				latestTarget := u.Targets.GetLatestTarget()
				if latestTarget.ID != u.ToTarget.ID {
					slog.Debug("Latest target is not the same as the ongoing update target, and should be cancelled", "ongoing_update_target_id", u.ToTarget.ID, "latest_target_id", latestTarget.ID)
					return ErrNewerVersionIsAvailable
				}
			}
		}
	} else {
		if s.ToVersion == -1 {
			if s.SyncCurrent {
				u.ToTarget = u.getSyncTarget()
				if u.ToTarget.ID == target.UnknownTarget.ID {
					return fmt.Errorf("could not find current target to be synced")
				}
			} else {
				u.ToTarget = u.Targets.GetLatestTarget()
				if u.ToTarget.ID == target.UnknownTarget.ID {
					return fmt.Errorf("could not find latest target")
				}
			}
			if s.MaxAttempts > 0 {
				count, err := update.CountFailedUpdates(u.Config.ComposeConfig(), u.ToTarget.ID)
				if err != nil {
					slog.Warn("Could not count failed updates for target", "target_id", u.ToTarget.ID, "error", err)
				} else {
					slog.Debug("Checking failed updates count for target", "target_id", u.ToTarget.ID, "count", count)
					if count >= s.MaxAttempts {
						slog.Info("Latest target installation attempts has reached the limit. Syncing current target", "latest_target_id", u.ToTarget.ID, "count", count)
						u.ToTarget = u.getSyncTarget()
					}
				}
			}
		} else {
			u.ToTarget = u.Targets.GetTargetByVersion(s.ToVersion)
			if u.ToTarget.ID == target.UnknownTarget.ID {
				return fmt.Errorf("could not find target with version %d", s.ToVersion)
			}
		}
	}
	return nil
}

func (u *UpdateContext) getSyncTarget() target.Target {
	// If the current target is still listed, then get its full info from the targets list
	// This is required in order to have all apps in updateCtx.ToTarget, as the running
	// target may have been installed while some apps were disabled
	targetFromList := u.Targets.GetTargetByVersion(u.FromTarget.Version)
	if targetFromList.ID != target.UnknownTarget.ID {
		return targetFromList
	} else {
		slog.Debug("current target not found in the targets list from the server", "current_target_id", u.FromTarget.ID)
		return u.FromTarget
	}
}

func (u *UpdateContext) isUpdateRequired(ctx context.Context) bool {
	if u.Type != UpdateTypeSync || u.Mode != UpdateModeNewUpdate {
		// Update may not be required only for the new sync update, otherwise an update is required
		return true
	}
	// If all current target apps are fetched, installed, and running, then no update is needed
	running, err := u.isTargetInSync(ctx)
	if err != nil {
		slog.Error("Failed to check if target is running; assume it is not", "error", err)
		return true
	}
	return !running
}

func (u *UpdateContext) isTargetInSync(ctx context.Context) (bool, error) {
	slog.Debug("Checking target", "target_id", u.ToTarget.ID)
	if u.checkAppDiff() {
		// There is some difference between a list of apps in FromTarget and ToTarget
		// and taking into account the enabled apps in the config
		return false, nil
	}
	var err error
	u.CurrentStatus, err = status.GetCurrentStatus(ctx, u.Config.ComposeConfig())
	if err != nil {
		return false, fmt.Errorf("error checking apps status: %w", err)
	}

	for _, appURI := range u.ToTarget.AppURIs() {
		appStatus, ok := u.CurrentStatus.AppStatuses[appURI]
		if !ok {
			slog.Debug("Required app is not present on the host", "app", appURI)
			return false, nil
		}
		if !appStatus.Fetched {
			slog.Debug("Required app is not fetched", "app", appURI)
			return false, nil
		}
		if !appStatus.Installed {
			slog.Debug("Required app is not installed", "app", appURI)
			return false, nil
		}
		if !appStatus.Running {
			slog.Debug("Required app is not running", "app", appURI)
			return false, nil
		}
	}
	slog.Info("Required applications are fetched, installed, and running")
	return true, nil
}

func (u *UpdateContext) checkAppDiff() bool {
	diffToCheck := []struct {
		diff     target.Apps
		diffType string
	}{
		{u.AppDiff.Add, "add"},
		{u.AppDiff.Remove, "remove"},
		{u.AppDiff.Update, "update"},
	}
	isDiffDetected := false
	for _, diffItem := range diffToCheck {
		for _, app := range diffItem.diff {
			slog.Debug("app list diff detected", "type", diffItem.diffType, "app", app.Name)
			isDiffDetected = true
		}
	}
	return isDiffDetected
}

func (u *UpdateContext) getAndSetCurrentTarget() error {
	u.FromTarget = target.UnknownTarget
	lastUpdate, err := update.GetLastSuccessfulUpdate(u.Config.ComposeConfig())
	if err != nil {
		return fmt.Errorf("no last successful update found: %w", err)
	}
	if target := u.Targets.GetTargetByID(lastUpdate.ClientRef); !target.IsUnknown() {
		target.ShortlistAppsByURI(lastUpdate.URIs)
		u.FromTarget = target
		return nil
	}
	slog.Debug("no target found in the targets list for the current target ID," +
		" trying to compose it from the last successful update")
	// Try to compose target from the last successful update info
	if target, err := getTargetOutOfUpdate(lastUpdate); err == nil {
		u.FromTarget = *target
		return nil
	} else {
		return fmt.Errorf("failed to compose current target from last successful update: %w", err)
	}
}

func (u *UpdateContext) getOngoingUpdateTarget() (*target.Target, error) {
	ongoingUpdate := u.UpdateRunner.Status()
	if target := u.Targets.GetTargetByID(ongoingUpdate.ClientRef); !target.IsUnknown() {
		target.ShortlistAppsByURI(ongoingUpdate.URIs)
		return &target, nil
	}
	slog.Debug("no target found in the targets list for the ongoing update target ID," +
		" trying to compose it from the ongoing update")
	// Try to compose target from the ongoing update info
	target, err := getTargetOutOfUpdate(&ongoingUpdate)
	if err != nil {
		return nil, fmt.Errorf("failed to compose ongoing update target from ongoing update: %w", err)
	}
	return target, nil
}

func getTargetOutOfUpdate(update *update.Update) (*target.Target, error) {
	version, err := extractTargetVersion(update.ClientRef)
	if err != nil {
		return nil, fmt.Errorf("failed to extract target version from last successful update ClientRef: %w", err)
	}
	apps, err := parseAppURIs(update.URIs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse app URIs from last successful update: %w", err)
	}
	return &target.Target{
		ID:      update.ClientRef,
		Apps:    apps,
		Version: version,
	}, nil
}

func parseAppURIs(appURIs []string) ([]target.App, error) {
	var apps []target.App
	for _, uri := range appURIs {
		appRef, err := compose.ParseAppRef(uri)
		if err != nil {
			return nil, fmt.Errorf("failed to parse app URI %q: %w", uri, err)
		}
		apps = append(apps, target.App{
			Name: appRef.Name,
			URI:  appRef.String(),
		})
	}
	return apps, nil
}

func extractTargetVersion(targetID string) (int, error) {
	parts := strings.Split(targetID, "-")
	if len(parts) < 2 {
		return -1, fmt.Errorf("invalid target ID format")
	}
	versionStr := parts[len(parts)-1]
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return -1, fmt.Errorf("invalid version in target ID: %w", err)
	}
	return version, nil
}
