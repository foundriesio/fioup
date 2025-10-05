// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"log/slog"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/pkg/client"
	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/target"
)

type (
	CheckOpts struct {
		TUF bool
	}
	CheckOpt func(*CheckOpts)
)

func WithTUF(enabled bool) CheckOpt {
	return func(o *CheckOpts) {
		o.TUF = enabled
	}
}

func Check(cfg *config.Config, options ...CheckOpt) (target.Targets, error) {
	opts := &CheckOpts{}
	for _, o := range options {
		o(opts)
	}

	var err error
	var dgClient *client.GatewayClient

	dgClient, err = client.NewGatewayClient(cfg, nil, "")
	if err != nil {
		return nil, err
	}
	var targetRepo target.Repo
	if opts.TUF {
		targetRepo, err = target.NewTufRepo(cfg, dgClient, cfg.GetHardwareID())
	} else {
		targetRepo, err = target.NewPlainRepo(dgClient, cfg.GetTargetsFilepath(), cfg.GetHardwareID())
	}
	if err != nil {
		return nil, err
	}
	var targets target.Targets
	targets, err = targetRepo.LoadTargets(false)
	if err != nil {
		slog.Debug("failed to load targets", "error", err)
	}

	var lastUpdate *update.Update
	lastUpdate, err = update.GetLastSuccessfulUpdate(cfg.ComposeConfig())
	if err != nil {
		slog.Debug("failed to get info about the last successful update", "error", err)
	}

	currentTarget := target.UnknownTarget
	if targets != nil && lastUpdate != nil {
		currentTarget = targets.GetTargetByID(lastUpdate.ClientRef)
		if currentTarget.IsUnknown() {
			slog.Debug("cannot find lastUpdate target in the target list", "target", lastUpdate.ClientRef)
		} else {
			currentTarget.ShortlistAppsByURI(lastUpdate.URIs)
		}
	}
	if !currentTarget.IsUnknown() {
		dgClient.UpdateHeaders(currentTarget.AppNames(), currentTarget.ID)
	}
	// Update targets by requesting the server for the latest targets metadata
	targets, err = targetRepo.LoadTargets(true)
	if err != nil {
		return nil, err
	}
	return targets, nil
}
