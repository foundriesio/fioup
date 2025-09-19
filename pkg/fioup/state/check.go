// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"fmt"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/pkg/fioup/target"
	"github.com/pkg/errors"
)

type Check struct {
	UpdateTargets  bool
	AllowNewUpdate bool
	Action         string
	AllowedStates  []update.State
	ToVersion      int
}

func (s *Check) Name() StateName { return "Checking" }
func (s *Check) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	var err error
	var actionType = ActionTypeResumeUpdate
	var targetProvider target.Provider

	targetProvider, err = target.NewTargetProvider(updateCtx.Config)
	if err != nil {
		return err
	}

	// Check if there is an ongoing update and set action type accordingly and fail if given action is not allowed
	updateCtx.UpdateRunner, err = update.GetCurrentUpdate(updateCtx.Config.ComposeConfig())
	if errors.Is(err, update.ErrUpdateNotFound) {
		if !s.AllowNewUpdate {
			return fmt.Errorf("no ongoing update to %s found;"+
				" please run %q or %q first", s.Action, "fioup update", "fioup fetch")
		}
		actionType = ActionTypeNewUpdate
		err = nil
	}
	if err != nil {
		return fmt.Errorf("failed to get info about current update: %w", err)
	}

	// Check if action is allowed at the current state of the update
	if actionType == ActionTypeResumeUpdate && len(s.AllowedStates) > 0 {
		currentState := updateCtx.UpdateRunner.Status().State
		if !currentState.IsIn(s.AllowedStates...) {
			return fmt.Errorf("cannot %s current update if it is in state %q", s.Action, currentState)
		}
	}

	// Get FromTarget: get last successful update to set FromTarget
	if lastUpdate, err := update.GetLastSuccessfulUpdate(updateCtx.Config.ComposeConfig()); err == nil {
		updateCtx.FromTarget, err = targetProvider.GetTargetByName(lastUpdate.ClientRef)
		if err != nil {
			return fmt.Errorf("could not find target of the last successful update: %w", err)
		}
		// TODO: update app list in FromTarget according to lastUpdate.URIs
	} else {
		updateCtx.FromTarget = target.NewUnknownTarget()
	}

	// Get ToTarget if resuming update
	if actionType == ActionTypeResumeUpdate {
		updateCtx.ToTarget, err = targetProvider.GetTargetByName(updateCtx.UpdateRunner.Status().ClientRef)
		if err != nil {
			// TODO: allow resuming update even if target is not found?
			return fmt.Errorf("could not find target of the ongoing update: %w", err)
		}
	} else {
		// Get ToTarget if starting new update
		if s.UpdateTargets {
			// Update targets if requested
			if err := targetProvider.UpdateTargets(updateCtx.Config); err != nil {
				return err
			}
		}
		if s.ToVersion == -1 {
			updateCtx.ToTarget, err = targetProvider.GetLatestTarget()
			if err != nil {
				return fmt.Errorf("could not find latest target: %w", err)
			}
		} else {
			updateCtx.ToTarget, err = targetProvider.GetTargetByVersion(s.ToVersion)
			if err != nil {
				return fmt.Errorf("could not find target with version %d: %w", s.ToVersion, err)
			}
		}
	}

	var updateType string
	if actionType == ActionTypeResumeUpdate {
		updateType = "Resuming"
	} else {
		updateType = "Start new"
	}
	fmt.Printf("\t\t%s update from %d to %d\n", updateType, updateCtx.FromTarget.Version(), updateCtx.ToTarget.Version())
	return nil
}
