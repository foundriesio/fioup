// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package fioup

import (
	"context"
	"fmt"
)

type (
	// StateMachine runs the OTA update states
	StateMachine struct {
		ctx    *UpdateContext
		states []State
	}
)

func NewStateMachine(cfg *Config, ctx *UpdateContext, states []State) (*StateMachine, error) {
	return &StateMachine{
		ctx:    ctx,
		states: states,
	}, nil
}

func (sm *StateMachine) Run(ctx context.Context) error {
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
