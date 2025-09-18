// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package fioup

import (
	"context"
	"fmt"
	"github.com/foundriesio/fioup/internal/targets"
	"github.com/foundriesio/fioup/pkg/fioup/config"
	"github.com/foundriesio/fioup/pkg/fioup/states"
	"github.com/foundriesio/fioup/pkg/fioup/target"
)

func Fetch(ctx context.Context, cfg *config.Config, toVersion int) error {
	var err error
	var targetProvider target.TargetProvider
	var fromTarget target.Target

	targetProvider, err = target.NewTargetProvider(cfg)
	if err != nil {
		return err
	}
	t, err := targets.GetCurrentTarget(cfg.GetDBPath())
	if err != nil {
		return err
	}
	fromTarget, err = target.NewTarget(t, cfg.GetEnabledApps())
	if err != nil {
		return err
	}
	stateMachine, err := states.NewStateMachine(cfg, &states.UpdateContext{
		Config:         cfg,
		TargetProvider: targetProvider,
		FromTarget:     fromTarget,
		ToVersion:      toVersion,
	}, []states.State{
		&states.CheckingState{UpdateTargets: false},
		&states.StagingState{},
		&states.FetchingState{},
	})
	if err != nil {
		return fmt.Errorf("failed to create state machine for fetching update: %w", err)
	}
	return stateMachine.Run(ctx)
}
