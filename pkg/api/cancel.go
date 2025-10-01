// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"context"

	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/pkg/config"
)

func Cancel(ctx context.Context, cfg *config.Config) (string, error) {
	currentUpdate, err := update.GetCurrentUpdate(cfg.ComposeConfig())
	if err != nil {
		return "", err
	}
	return currentUpdate.Status().ClientRef, currentUpdate.Cancel(ctx)
}
