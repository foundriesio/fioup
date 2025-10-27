// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
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

	// Get FromTarget: get last successful update to set FromTarget
	if fromTarget, err := getCurrentTarget(updateCtx.Config.ComposeConfig()); err == nil {
		updateCtx.FromTarget = *fromTarget
		updateCtx.Client.UpdateHeaders(updateCtx.FromTarget.AppNames(), updateCtx.FromTarget.ID)
	} else {
		slog.Debug("failed to determine the current target, consider it as unknown", "error", err)
		updateCtx.FromTarget = target.UnknownTarget
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

	if updateCtx.Mode == UpdateModeResume {
		// Get ToTarget if resuming update
		ongoingUpdate := updateCtx.UpdateRunner.Status()
		target, err := getTargetOutOfUpdate(&ongoingUpdate)
		if err != nil {
			return ErrInvalidOngoingUpdate
		}
		updateCtx.ToTarget = *target
		if s.ToVersion != -1 {
			// make sure ToVersion matches the ongoing update, otherwise fail
			if updateCtx.ToTarget.Version != s.ToVersion {
				return fmt.Errorf("cannot start or resume update to version %d since there is an ongoing update to version %d",
					s.ToVersion, updateCtx.ToTarget.Version)
			}
		} else {
			if s.RequireLatest {
				// When running in daemon mode, do not resume the ongoing update if the latest target has changed
				// Calling code is expected to cancel the ongoing update in this case
				latestTarget := targets.GetLatestTarget()
				if latestTarget.ID != updateCtx.ToTarget.ID {
					slog.Debug("Latest target is not the same as the ongoing update target, and should be cancelled", "ongoing_update_target_id", updateCtx.ToTarget.ID, "latest_target_id", latestTarget.ID)
					return ErrNewerVersionIsAvailable
				}
			}
		}
	} else {
		if s.ToVersion == -1 {
			if s.SyncCurrent {
				updateCtx.ToTarget = updateCtx.FromTarget
				if updateCtx.ToTarget.ID == target.UnknownTarget.ID {
					return fmt.Errorf("could not find current target to be synced")
				}
			} else {
				updateCtx.ToTarget = targets.GetLatestTarget()
				if updateCtx.ToTarget.ID == target.UnknownTarget.ID {
					return fmt.Errorf("could not find latest target: %w", err)
				}
			}

			if s.MaxAttempts > 0 {
				count, err := update.CountFailedUpdates(updateCtx.Config.ComposeConfig(), updateCtx.ToTarget.ID)
				if err != nil {
					slog.Warn("Could not count failed updates for target", "target_id", updateCtx.ToTarget.ID, "error", err)
				} else {
					slog.Debug("Checking failed updates count for target", "target_id", updateCtx.ToTarget.ID, "count", count)
					if count >= s.MaxAttempts {
						slog.Info("Latest target installation attempts has reached the limit. Syncing current target", "latest_target_id", updateCtx.ToTarget.ID, "count", count)
						updateCtx.ToTarget = updateCtx.FromTarget
					}
				}
			}

			// If an update is not forced, and the target is the same as the current one,
			// then check if the system is in sync with the target. If it is, then skip the update.
			if !s.Force && updateCtx.ToTarget.ID == updateCtx.FromTarget.ID {
				updateCtx.ToTarget.ShortlistApps(updateCtx.Config.GetEnabledApps())
				running, err := isTargetRunning(ctx, updateCtx)
				if err != nil {
					slog.Error("Failed to check if target is running", "error", err)
				}
				if running {
					slog.Info("Skipping running target", "target_id", updateCtx.ToTarget.ID)
					return ErrCheckNoUpdate
				}
			}
		} else {
			updateCtx.ToTarget = targets.GetTargetByVersion(s.ToVersion)
			if updateCtx.ToTarget.ID == target.UnknownTarget.ID {
				return fmt.Errorf("could not find target with version %d: %w", s.ToVersion, err)
			}
		}
	}
	updateCtx.ToTarget.ShortlistApps(updateCtx.Config.GetEnabledApps())
	if updateCtx.ToTarget.ID == updateCtx.FromTarget.ID {
		updateCtx.Type = UpdateTypeSync
	} else {
		updateCtx.Type = UpdateTypeUpdate
	}
	updateCtx.AppDiff.Add, updateCtx.AppDiff.Remove, updateCtx.AppDiff.Sync, updateCtx.AppDiff.Update = updateCtx.FromTarget.Diff(&updateCtx.ToTarget)
	if updateCtx.UpdateRunner != nil {
		updateCtx.InitializedAt = updateCtx.UpdateRunner.Status().InitTime
	}
	return nil
}

func isTargetRunning(ctx context.Context, updateContext *UpdateContext) (bool, error) {
	slog.Debug("Checking target", "target_id", updateContext.ToTarget.ID)
	if len(updateContext.ToTarget.Apps) == 0 {
		slog.Debug("No required apps to check")
		return true, nil
	}

	if isSublist(updateContext.FromTarget.AppURIs(), updateContext.ToTarget.AppURIs()) {
		slog.Debug("Installed applications match selected target apps")
		status, err := compose.CheckAppsStatus(ctx, updateContext.Config.ComposeConfig(), updateContext.ToTarget.AppURIs())
		if err != nil {
			return false, fmt.Errorf("error checking apps status: %w", err)
		}

		if status.AreRunning() {
			slog.Info("Required applications are running")
			return true, nil
		} else {
			slog.Info("Required applications are not running", "apps_not_running", status.NotRunningApps)
			return false, nil
		}
	} else {
		slog.Debug("Installed applications list do not contain all target apps")
		return false, nil
	}
}

func isSublist[S ~[]E, E comparable](mainList, sublist S) bool {
	if len(sublist) > len(mainList) {
		return false
	}
	for _, subElem := range sublist {
		if !slices.Contains(mainList, subElem) {
			return false
		}
	}
	return true
}

func getCurrentTarget(cfg *compose.Config) (*target.Target, error) {
	lastUpdate, err := update.GetLastSuccessfulUpdate(cfg)
	if err != nil {
		return nil, fmt.Errorf("no last successful update found: %w", err)
	}
	return getTargetOutOfUpdate(lastUpdate)
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
