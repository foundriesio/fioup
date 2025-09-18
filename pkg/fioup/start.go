// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package fioup

import (
	"context"
	"fmt"
	"github.com/foundriesio/fioup/internal/targets"
)

func Start(ctx context.Context, cfg *Config) error {
	var err error
	var targetProvider TargetProvider
	var fromTarget Target

	targetProvider, err = NewTargetProvider(cfg)
	if err != nil {
		return err
	}
	t, err := targets.GetCurrentTarget(cfg.GetDBPath())
	if err != nil {
		return err
	}
	fromTarget, err = NewTarget(t, cfg.GetEnabledApps())
	if err != nil {
		return err
	}
	stateMachine, err := NewStateMachine(cfg, &UpdateContext{
		Config:         cfg,
		TargetProvider: targetProvider,
		FromTarget:     fromTarget,
	}, []State{
		&CheckingState{UpdateTargets: false},
		&StagingState{},
		&FetchingState{},
		&InstallingState{},
		&StartingState{},
	})
	if err != nil {
		return fmt.Errorf("failed to create state machine for starting update: %w", err)
	}
	return stateMachine.Run(ctx)
}
