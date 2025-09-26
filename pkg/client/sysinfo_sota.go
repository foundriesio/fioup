// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package client

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"

	"github.com/foundriesio/fioconfig/sotatoml"
)

// uploadSotaToml uploads the *combined* sota TOML configuration IFF its changed.
func (c *GatewayClient) uploadSotaToml() error {
	if c.sotaToReport == nil {
		slog.Debug("Sota TOML has not changed")
		return nil
	}
	res, err := c.Put("/system_info/config", c.sotaToReport)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("unable to update config info. HTTP_%d: %s", res.StatusCode, res.String())
	}

	toSave := c.sotaToReport
	c.sotaToReport = nil // no matter what happens below - don't re-publish

	if err := os.WriteFile(c.lastSotaFile, toSave, 0o744); err != nil {
		slog.Error("Unable to save sota-toml that was pushed to server", "error", err)
	}

	return nil
}

func (c *GatewayClient) initSota(config *sotatoml.AppConfig) {
	tree, err := config.CombinedConfig()
	if err != nil {
		slog.Error("Unable to parse TOML files. Will not publish to server", "error", err)
		return
	}

	current, err := tree.Marshal()
	if err != nil {
		slog.Error("Unable to marshall TOML files. Will not publish to server", "error", err)
		return
	}

	reported, err := os.ReadFile(c.lastSotaFile)
	if err == nil && bytes.Equal(reported, current) {
		return
	}
	c.sotaToReport = current // Lets UploadSotaToml know to publish
}
