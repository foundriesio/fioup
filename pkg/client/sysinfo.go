// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package client

// PutSysInfo sends the "system-info" API data to the gateway. It only sends
// each piece of data if it has changed since the last time this function
// was invoked.
func (c *GatewayClient) PutSysInfo() error {
	return nil
}
