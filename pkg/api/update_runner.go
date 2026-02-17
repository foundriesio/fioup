// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/foundriesio/fioup/internal/db"
	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/pkg/client"
	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/state"
	"github.com/foundriesio/fioup/pkg/target"
)

type (
	UpdateInfo = state.UpdateInfo
	// UpdateRunner runs the OTA update states
	UpdateRunner struct {
		opts   *UpdateRunnerOpts
		ctx    *state.UpdateContext
		states []state.ActionState
	}
	UpdateRunnerOpts struct {
		EventSender      *events.EventSender
		GatewayClient    *client.GatewayClient
		PreStateHandler  PreStateHandler
		PostStateHandler PostStateHandler
	}
	UpdateRunnerOpt func(*UpdateRunnerOpts)

	UpdateMode       = state.UpdateMode
	StateName        = state.ActionName
	PreStateHandler  func(StateName, *UpdateInfo)
	PostStateHandler func(StateName, *UpdateInfo)
)

func newUpdateRunner(states []state.ActionState, options ...UpdateRunnerOpt) *UpdateRunner {
	opts := &UpdateRunnerOpts{}
	for _, o := range options {
		o(opts)
	}
	return &UpdateRunner{
		opts: opts,
		ctx: &state.UpdateContext{
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

	if err := db.InitializeDatabase(cfg.GetDBPath()); err != nil {
		return err
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

	sm.ctx.TotalStates = len(sm.states)
	sm.ctx.CurrentStateNum = 1
	for _, s := range sm.states {
		sm.ctx.CurrentState = s.Name()
		if sm.opts.PreStateHandler != nil {
			sm.opts.PreStateHandler(s.Name(), &sm.ctx.UpdateInfo)
		}
		err := s.Execute(ctx, sm.ctx)
		if err != nil {
			return fmt.Errorf("failed at state %s: %w", s.Name(), err)
		}
		if sm.opts.PostStateHandler != nil {
			sm.opts.PostStateHandler(s.Name(), &sm.ctx.UpdateInfo)
		}
		sm.ctx.CurrentStateNum++
	}
	if err := gwClient.ReportAppStates(ctx, cfg.ComposeConfig()); err != nil {
		slog.Debug("failed to report apps states", "error", err)
	}
	return nil
}
