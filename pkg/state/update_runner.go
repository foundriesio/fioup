// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/foundriesio/fioup/internal/events"
	internal "github.com/foundriesio/fioup/internal/update"
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
	if err := internal.InitializeDatabase(cfg.GetDBPath()); err != nil {
		return err
	}
	gwClient, err := client.NewGatewayClient(cfg, nil, "")
	if err != nil {
		return err
	}
	sm.ctx.Config = cfg
	sm.ctx.EventSender = &EventSender{
		dbPath:   cfg.GetDBPath(),
		gwClient: gwClient,
	}
	stateCounter := 1
	for _, s := range sm.states {
		sm.ctx.CurrentState = s.Name()
		fmt.Printf("[%d/5] %s:", stateCounter, s.Name())
		err := s.Execute(ctx, sm.ctx)
		sm.ctx.EventSender.FlushEvents()
		if err != nil {
			return fmt.Errorf("failed at state %s: %w", s.Name(), err)
		}
		stateCounter++
	}
	return nil
}

func (u *UpdateContext) SendEvent(event events.EventTypeValue, success ...bool) {
	if err := u.EventSender.SendEvent(event, u.UpdateRunner.Status().ID, u.ToTarget, success...); err != nil {
		slog.Error("failed to send event", "event", event, "err", err)
	}
}

func (s *EventSender) SendEvent(eventType events.EventTypeValue, updateID string, toTarget target.Target, success ...bool) error {
	var completionStatus *bool
	if len(success) > 0 {
		completionStatus = &success[0]
	}
	if eventType == events.InstallationCompleted && completionStatus != nil && *completionStatus {
		// Update list of apps and target ID if update is successful
		s.gwClient.UpdateHeaders(toTarget.AppNames(), toTarget.ID)
	}
	evt := events.NewEvent(eventType, "", completionStatus, updateID, toTarget.ID, toTarget.Version)
	return events.SaveEvent(s.dbPath, &evt[0])
}

func (s *EventSender) FlushEvents() {
	if err := events.FlushEvents(s.dbPath, s.gwClient); err != nil {
		slog.Error("failed to flush events", "err", err)
	}
}
