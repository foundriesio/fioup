// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"fmt"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
)

type Install struct{}

func (s *Install) Name() StateName { return "Installing" }
func (s *Install) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	// Stop apps being updated before installing their updates
	if err := compose.StopApps(ctx, updateCtx.Config.ComposeConfig(), updateCtx.FromTarget.Apps()); err != nil {
		return err
	}
	fmt.Printf("\n")
	err := updateCtx.UpdateRunner.Install(ctx, compose.WithInstallProgress(update.GetInstallProgressPrinter()))
	return err
}
