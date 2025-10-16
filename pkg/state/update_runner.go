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
	UpdateRunnerOpts struct {
		EventSender   *events.EventSender
		GatewayClient *client.GatewayClient
	}
	UpdateRunnerOpt func(*UpdateRunnerOpts)
)

func NewUpdateRunner(states []ActionState, options ...UpdateRunnerOpt) *UpdateRunner {
	opts := &UpdateRunnerOpts{}
	for _, o := range options {
		o(opts)
	}
	return &UpdateRunner{
		ctx: &UpdateContext{
			EventSender: opts.EventSender,
			Client:      opts.GatewayClient,
		},
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

	var err error
	gwClient := sm.ctx.Client
	if gwClient == nil {
		gwClient, err = client.NewGatewayClient(cfg, nil, "")
		if err != nil {
			return fmt.Errorf("failed to create gateway client: %w", err)
		}
		sm.ctx.Client = gwClient
	}
	eventSender := sm.ctx.EventSender
	if eventSender == nil {
		eventSender, err = events.NewEventSender(cfg, gwClient)
		if err != nil {
			return err
		}
		sm.ctx.EventSender = eventSender
		eventSender.Start()
		defer eventSender.Stop()
	}

	// TODO: add an option to turn on/off sysinfo upload
	if err := gwClient.PutSysInfo(); err != nil {
		slog.Error("Unable to upload sysinfo", "error", err)
	}
	if err := gwClient.ReportAppStates(ctx, cfg.ComposeConfig()); err != nil {
		slog.Debug("failed to report apps states", "error", err)
	}

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
	if err := gwClient.ReportAppStates(ctx, cfg.ComposeConfig()); err != nil {
		slog.Debug("failed to report apps states", "error", err)
	}
	return nil
}

func (u *UpdateContext) SendEvent(event events.EventTypeValue, success ...bool) {
	if err := u.EventSender.EnqueueEvent(event, u.UpdateRunner.Status().ID, u.ToTarget, success...); err != nil {
		slog.Error("failed to send event", "event", event, "err", err)
	}
}
