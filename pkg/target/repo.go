// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package target

type (
	Repo interface {
		LoadTargets(update bool) (Targets, error)
	}
)
