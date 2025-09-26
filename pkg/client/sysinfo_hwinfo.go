// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package client

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"

	"github.com/foundriesio/fioconfig/transport"
)

// uploadHwinfo uploads the output of `lshw` IFF its changed.
func (c *GatewayClient) uploadHwinfo() error {
	if c.hwinfoToReport == nil {
		slog.Debug("Hardware-info has not changed")
		return nil
	}
	headers := map[string]string{"Content-type": "application/json"}
	url := c.BaseURL.JoinPath("/system_info").String()
	res, err := transport.HttpDo(c.HttpClient, http.MethodPut, url, headers, c.hwinfoToReport)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("unable to update hwinfo info. HTTP_%d: %s", res.StatusCode, res.String())
	}

	toSave := c.hwinfoToReport
	c.hwinfoToReport = nil // no matter what happens below - don't re-publish

	if err := os.WriteFile(c.lastHwinfoFile, toSave, 0o744); err != nil {
		slog.Error("Unable to save hwinfo that was pushed to server", "error", err)
	}

	return nil
}

func (c *GatewayClient) initHwinfo() {
	path, err := exec.LookPath("lshw")
	if err != nil {
		slog.Debug("lshw not available. Will not publish hardware-info")
	}

	cmd := exec.Command(path, "-json", "-notime")
	output, err := cmd.Output()
	if err != nil {
		slog.Error("Unable to run lshw", "error", err)
		return
	}

	reported, err := os.ReadFile(c.lastHwinfoFile)
	if err == nil && bytes.Equal(reported, output) {
		return
	}
	c.hwinfoToReport = output // Lets uploadHwInfo know to publish
}
