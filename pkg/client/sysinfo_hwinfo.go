// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package client

import (
	"bytes"
	"encoding/json"
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

// scrubHwInfo looks at fields known to change for each call of `lshw` and cleans
// them up so that we don't report every single change to the backend but only
// major ones.
func scrubHwinfo(lshwBytes []byte) []byte {
	var hwInfo map[string]any
	if err := json.Unmarshal(lshwBytes, &hwInfo); err != nil {
		slog.Error("Unexpected error unmarshalling lshw output", "error", err)
	}

	// cpu frequency (cpu "size") is always changing. The actual capacity is
	// the more interesting value, so just remove this attribute
	children := hwInfo["children"].([]any)
	for _, child := range children {
		child := child.(map[string]any)
		if child["id"].(string) == "core" {
			children := child["children"].([]any)
			for _, child := range children {
				child := child.(map[string]any)
				if child["id"].(string) == "cpu" {
					if _, ok := child["size"]; ok {
						slog.Debug("Deleting CPU frequency from lshw output", "freq", child["size"])
						delete(child, "size")
						break
					}
				}
			}
			break
		}
	}

	output, err := json.Marshal(hwInfo)
	if err != nil {
		slog.Error("Unexpected error marshalling lshw data", "error", err)
		output = lshwBytes // this is the best we can do - don't scrub, the output
	}

	return output
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
	output = scrubHwinfo(output)

	reported, err := os.ReadFile(c.lastHwinfoFile)
	if err == nil && bytes.Equal(reported, output) {
		return
	}
	c.hwinfoToReport = output // Lets uploadHwInfo know to publish
}
