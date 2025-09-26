// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package client

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/foundriesio/fioup/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestGatewayClient_uploadSotaToml(t *testing.T) {
	// There's not much we can really test but we can make sure the caching logic is correct
	tmpdir := t.TempDir()
	sotaFile := filepath.Join(tmpdir, "sota.toml")
	sota := fmt.Sprintf(`
[tls]
server = "https://example.com:8443"

[storage]
path = "%s"

[pacman]
reset_apps_root = "%s"
compose_apps_root = "%s"
	`, tmpdir, tmpdir, tmpdir)

	require.Nil(t, os.WriteFile(sotaFile, []byte(sota), 0o744))

	cfg, err := config.NewConfig([]string{tmpdir})
	require.Nil(t, err)

	gw := GatewayClient{}
	gw.initSota(cfg.TomlConfig())
	require.NotNil(t, gw.sotaToReport)
	gw.sotaToReport = nil
	require.Nil(t, gw.uploadSotaToml())
}
