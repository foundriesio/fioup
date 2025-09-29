// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"fmt"
	"strings"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/pkg/client"
	"github.com/foundriesio/fioup/pkg/target"
	"github.com/pkg/errors"
)

type (
	Check struct {
		UpdateTargets  bool
		AllowNewUpdate bool
		Action         string
		AllowedStates  []update.State
		ToVersion      int
	}
)

const (
	updateModeStarting = "starting"
	updateModeResuming = "resuming"
)

func (s *Check) Name() ActionName { return "Checking" }
func (s *Check) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	var err error
	var updateMode string

	// Check if there is an ongoing update and set action type accordingly and fail if given action is not allowed
	updateCtx.UpdateRunner, err = update.GetCurrentUpdate(updateCtx.Config.ComposeConfig())
	if errors.Is(err, update.ErrUpdateNotFound) {
		if !s.AllowNewUpdate {
			return fmt.Errorf("no ongoing update to %s found;"+
				" please run %q or %q first", s.Action, "fioup update", "fioup fetch")
		}
		updateMode = updateModeStarting
		err = nil
	} else {
		updateMode = updateModeResuming
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
	var gwClient *client.GatewayClient
	gwClient, err = client.NewGatewayClient(updateCtx.Config, updateCtx.FromTarget.AppNames(), updateCtx.FromTarget.ID)
	if err != nil {
		return err
	}
	targetRepo, err = target.NewPlainRepo(gwClient, updateCtx.Config.GetTargetsFilepath(), updateCtx.Config.GetHardwareID())
	if err != nil {
		return err
	}
	targets, err := targetRepo.LoadTargets(s.UpdateTargets)
	if err != nil {
		return err
	}

	// Get FromTarget: get last successful update to set FromTarget
	if lastUpdate, err := update.GetLastSuccessfulUpdate(updateCtx.Config.ComposeConfig()); err == nil {
		updateCtx.FromTarget = targets.GetTargetByID(lastUpdate.ClientRef)
		if updateCtx.FromTarget.ID == target.UnknownTarget.ID {
			return fmt.Errorf("could not find target of the last successful update: %w", err)
		}
		updateCtx.FromTarget.ShortlistAppsByURI(lastUpdate.URIs)
	} else {
		updateCtx.FromTarget = target.UnknownTarget
	}

	if updateMode == updateModeResuming {
		// Get ToTarget if resuming update
		updateCtx.ToTarget = targets.GetTargetByID(updateCtx.UpdateRunner.Status().ClientRef)
		if updateCtx.ToTarget.ID == target.UnknownTarget.ID {
			// TODO: allow resuming update even if target is not found?
			return fmt.Errorf("could not find target of the ongoing update: %w", err)
		}
		if s.ToVersion != -1 {
			// make sure ToVersion matches the ongoing update, otherwise fail
			if updateCtx.ToTarget.Version != s.ToVersion {
				return fmt.Errorf("cannot start or resume update to version %d since there is an ongoing update to version %d",
					s.ToVersion, updateCtx.ToTarget.Version)
			}
		}
	} else {
		if s.ToVersion == -1 {
			updateCtx.ToTarget = targets.GetLatestTarget()
			if updateCtx.ToTarget.ID == target.UnknownTarget.ID {
				return fmt.Errorf("could not find latest target: %w", err)
			}
		} else {
			updateCtx.ToTarget = targets.GetTargetByVersion(s.ToVersion)
			if updateCtx.ToTarget.ID == target.UnknownTarget.ID {
				return fmt.Errorf("could not find target with version %d: %w", s.ToVersion, err)
			}
		}
	}
	updateCtx.ToTarget.ShortlistApps(updateCtx.Config.GetEnabledApps())

	fmt.Printf("\t\t%s update from %d [%s] to %d [%s]\n",
		updateMode, updateCtx.FromTarget.Version, strings.Join(updateCtx.FromTarget.AppNames(), ","),
		updateCtx.ToTarget.Version, strings.Join(updateCtx.ToTarget.AppNames(), ","))
	return nil
}
