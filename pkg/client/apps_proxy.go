// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package client

import (
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/fioconfig/transport"
)

type AppsProxy struct {
	Url    string
	CaCert *x509.CertPool
}

// Configurre sets environment variables required by composectl to use the proxy
func (p AppsProxy) Configure(cfg *compose.Config) error {
	proxyURL, err := url.Parse(p.Url)
	if err != nil {
		return err
	}
	cfg.ProxyURL = proxyURL
	cfg.ProxyCerts = p.CaCert
	return nil
}

// Unconfigure unsets environment variables used for the apps proxy
func (p AppsProxy) Unconfigure(cfg *compose.Config) {
	cfg.ProxyURL = nil
	cfg.ProxyCerts = nil
}

// AppsProxyUrl will look to see if an apps proxy server is configured. If so,
// it will request a proxy URL from that resource and return it.
// Returns nil if no proxy server is configured or an error if there
// was an issue requesting the URL.
func (c *GatewayClient) AppsProxyUrl() (*AppsProxy, error) {
	if len(c.proxyAppsUrl) == 0 {
		return nil, nil
	}

	resp, err := transport.HttpDo(c.HttpClient, http.MethodPost, c.proxyAppsUrl, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to request apps proxy URL: %w", err)
	} else if resp.StatusCode != 201 {
		return nil, fmt.Errorf("unexpected response code %d requesting apps proxy URL: %s", resp.StatusCode, resp.String())
	}

	return &AppsProxy{
		Url:    resp.String(),
		CaCert: c.proxyAppsCa,
	}, nil
}
