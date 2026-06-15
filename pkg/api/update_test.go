// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithFetchWorkers(t *testing.T) {
	tests := []struct {
		name string
		opts []UpdateOpt
		want int
	}{
		{
			name: "unset defaults to zero (composeapp applies its own default)",
			opts: nil,
			want: 0,
		},
		{
			name: "explicit worker count is propagated",
			opts: []UpdateOpt{WithFetchWorkers(5)},
			want: 5,
		},
		{
			name: "last option wins",
			opts: []UpdateOpt{WithFetchWorkers(2), WithFetchWorkers(7)},
			want: 7,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getUpdateOpts(tc.opts...)
			assert.Equal(t, tc.want, got.FetchWorkers)
		})
	}
}
