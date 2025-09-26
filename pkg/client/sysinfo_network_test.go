// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ipInfo(t *testing.T) {
	// we can't really test/validate ipinfo well, so:
	//  * make sure it doesn't fail
	//  * make sure we don't try to report if it hasn't changed
	var err error
	info := netInfo{}
	info.Host, err = os.Hostname()
	require.Nil(t, err)
	info.Ip, info.Mac, err = ipInfo()
	require.Nil(t, err)
	// We can't really test these values so just dump them
	t.Logf("Local IPv4: %s", info.Ip)
	t.Logf("Mac addr: %s", info.Mac)

	// Now cache this info
	infoBytes, err := json.Marshal(info)
	require.Nil(t, err)

	tmpDir := t.TempDir()
	gw := GatewayClient{
		lastNetInfoFile: filepath.Join(tmpDir, "ipinfo"),
	}

	// If our caching logic doesn't work - this will crash with a nil pointer
	// trying to talk to the device-gateway
	require.Nil(t, os.WriteFile(gw.lastNetInfoFile, infoBytes, 0o740))
	require.Nil(t, gw.uploadNetInfo())
}
