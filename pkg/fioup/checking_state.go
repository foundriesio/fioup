// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package fioup

import (
	"context"
	"fmt"
	"github.com/foundriesio/composeapp/pkg/update"
)

type CheckingState struct {
	UpdateTargets bool
}

func (s *CheckingState) Name() StateName { return Checking }
func (s *CheckingState) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	var err error
	var updateType string

	// FromTarget: get last successful update to set FromTarget
	if lastUpdate, err := update.GetLastSuccessfulUpdate(updateCtx.Config.ComposeConfig()); err == nil {
		updateCtx.FromTarget, err = updateCtx.TargetProvider.GetTargetByName(lastUpdate.ClientRef)
		if err != nil {
			return fmt.Errorf("could not find target of the last successful update: %w", err)
		}
		// TODO: update app list in FromTarget according to lastUpdate.URIs
	}

	updateCtx.Runner, _ = update.GetCurrentUpdate(updateCtx.Config.ComposeConfig())
	// ToTarget: get it from ongoing update if any or from TargetProvider
	if updateCtx.Runner == nil || updateCtx.Runner.Status().State == update.StateCreated {
		// No update in progress, figure out ToTarget based on list of targets provided by TargetProvider
		if s.UpdateTargets {
			// Update targets if requested
			if err := updateCtx.TargetProvider.UpdateTargets(updateCtx.Config); err != nil {
				return err
			}
		}
		if updateCtx.ToVersion == -1 {
			updateCtx.ToTarget, err = updateCtx.TargetProvider.GetLatestTarget()
		} else {
			updateCtx.ToTarget, err = updateCtx.TargetProvider.GetTargetByVersion(updateCtx.ToVersion)
		}
		updateType = "starting new"
	} else {
		updateCtx.ToTarget, err = updateCtx.TargetProvider.GetTargetByName(updateCtx.Runner.Status().ClientRef)
		// TODO: update app list in ToTarget according to runner.Status().URIs
		updateType = "resuming"
	}
	if err != nil {
		return err
	}

	updateCtx.CurrentState = Checked
	fmt.Printf("\t\t%s update from %d to %d\n", updateType, updateCtx.FromTarget.Version(), updateCtx.ToTarget.Version())
	return nil
}
