// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/foundriesio/fioup/internal/db"
	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/pkg/client"
	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/target"
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

func (sm *UpdateRunner) GetFromTarget() target.Target {
	return sm.ctx.FromTarget
}

func (sm *UpdateRunner) Run(ctx context.Context, cfg *config.Config) error {
	if err := db.InitializeDatabase(cfg.GetDBPath()); err != nil {
		return err
	}
	sm.ctx.Config = cfg

	client, err := client.NewGatewayClient(cfg, nil, "")
	if err != nil {
		return err
	}
	eventSender, err := events.NewEventSender(cfg, client)
	if err != nil {
		return err
	}
	eventSender.Start()
	defer eventSender.Stop()

	// TODO: add an option to turn on/off sysinfo upload
	if err := client.PutSysInfo(); err != nil {
		slog.Error("Unable to upload sysinfo", "error", err)
	}

	sm.ctx.EventSender = eventSender
	sm.ctx.Client = client

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

func (u *UpdateContext) SendEvent(event events.EventTypeValue, success ...bool) {
	if err := u.EventSender.EnqueueEvent(event, u.UpdateRunner.Status().ID, u.ToTarget, success...); err != nil {
		slog.Error("failed to send event", "event", event, "err", err)
	}
}
