// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package fioup

import (
	"context"
	"fmt"
	"github.com/foundriesio/fioup/internal/targets"
)

func Fetch(ctx context.Context, cfg *Config, toVersion int) error {
	var err error
	var targetProvider TargetProvider
	var fromTarget Target

	targetProvider, err = NewUnsafeTargetProvider(cfg)
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
		ToVersion:      toVersion,
	}, []State{
		&CheckingState{UpdateTargets: false},
		&StagingState{},
		&FetchingState{},
	})
	if err != nil {
		return fmt.Errorf("failed to create state machine for fetching update: %w", err)
	}
	return stateMachine.Run(ctx)
}
