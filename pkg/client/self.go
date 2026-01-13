// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package client

import (
	"time"
)

type DgTimeStamp float32

func (ts DgTimeStamp) AsTime() time.Time {
	return time.Unix(int64(ts), 0)
}

type Device struct {
	Factory   string      `json:"factory"`
	RepoId    string      `json:"repo_id"`
	Uuid      string      `json:"uuid"`
	Name      string      `json:"name"`
	CreatedAt DgTimeStamp `json:"created_at"`
	LastSeen  DgTimeStamp `json:"last_seen"`
	PubKey    string      `json:"pubkey"`
	Tag       string      `json:"tag"`
}

func (c *GatewayClient) Self() (*Device, error) {
	var d Device
	return &d, c.getJson("/device", &d)
}
