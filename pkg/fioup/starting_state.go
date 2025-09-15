// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package fioup

import (
	"context"
	"github.com/foundriesio/composeapp/pkg/update"
)

type StartingState struct{}

func (s *StartingState) Name() StateName { return Starting }
func (s *StartingState) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	err := updateCtx.Runner.Start(ctx)
	if err == nil {
		return updateCtx.Runner.Complete(ctx, update.CompleteWithPruning())
	}
	return err
}
