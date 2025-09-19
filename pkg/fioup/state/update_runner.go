// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"fmt"

	"github.com/foundriesio/fioup/pkg/fioup/config"
)

type (
	// UpdateRunner runs the OTA update states
	UpdateRunner struct {
		ctx    *UpdateContext
		states []ActionState
	}
)

func NewUpdateRunner(states []ActionState) *UpdateRunner {
	return &UpdateRunner{
		ctx:    &UpdateContext{},
		states: states,
	}
}

func (sm *UpdateRunner) Run(ctx context.Context, cfg *config.Config) error {
	sm.ctx.Config = cfg
	stateCounter := 1
	for _, s := range sm.states {
		sm.ctx.CurrentState = s.Name()
		fmt.Printf("[%d/5] %s:", stateCounter, s.Name())
		err := s.Execute(ctx, sm.ctx)
		if err != nil {
			return fmt.Errorf("failed at state %s: %w", s.Name(), err)
		}
		stateCounter++
	}
	return nil
}
