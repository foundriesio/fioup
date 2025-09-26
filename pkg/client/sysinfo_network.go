// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
)

type netInfo struct {
	Host string `json:"hostname"`
	Mac  string `json:"mac"`
	Ip   string `json:"local_ipv4"`
}

// UploadNetInfo uploads local ipv4 info to the gateway IFF its changed.
func (c GatewayClient) uploadNetInfo() error {
	var err error
	info := netInfo{}
	info.Host, err = os.Hostname()
	if err != nil {
		return err
	}

	info.Ip, info.Mac, err = ipInfo()
	if err != nil {
		return err
	}
	newBytes, err := json.Marshal(info)
	if err != nil {
		slog.Error("unexpected error marshalling net-info data", "error", err)
	}

	if lastInfoBytes, err := os.ReadFile(c.lastNetInfoFile); err == nil {
		if bytes.Equal(newBytes, lastInfoBytes) {
			slog.Debug("Net-info has not changed")
			return nil
		}
	}
	slog.Debug("Net-info has change, uploading to server", "value", info)

	res, err := c.Put("/system_info/network", info)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("unable to update network info. HTTP_%d: %s", res.StatusCode, res.String())
	}

	if err := os.WriteFile(c.lastNetInfoFile, newBytes, 0o744); err != nil {
		slog.Error("Unable to save net-info bytes pushed to server", "error", err)
	}
	return nil
}

func ipInfo() (string, string, error) {
	netInfo, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return "", "", err
	}

	for i, line := range strings.Split(string(netInfo), "\n") {
		if i > 0 {
			parts := strings.Fields(line)
			if len(parts) > 4 && parts[1] == "00000000" {
				intf, err := net.InterfaceByName(parts[0])
				if err != nil {
					return "", "", fmt.Errorf("unable to lookup default interface(%s): %w", parts[0], err)
				}
				addrs, err := intf.Addrs()
				if err != nil {
					return "", "", fmt.Errorf("unable to lookup IP of interface(%s): %w", parts[0], err)
				}
				return addrs[0].String(), intf.HardwareAddr.String(), nil
			}
		}
	}

	return "", "", errors.New("could not find default network interface")
}
