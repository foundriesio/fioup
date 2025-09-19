// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"github.com/foundriesio/composeapp/pkg/update"
)

type Start struct{}

func (s *Start) Name() StateName { return "Starting" }
func (s *Start) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	err := updateCtx.UpdateRunner.Start(ctx)
	if err == nil {
		return updateCtx.UpdateRunner.Complete(ctx, update.CompleteWithPruning())
	}
	return err
}
