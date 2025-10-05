// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package client

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/foundriesio/fioconfig/transport"
	"github.com/foundriesio/fioup/pkg/config"
)

type (
	GatewayClient struct {
		BaseURL    *url.URL
		HttpClient *http.Client
		Headers    map[string]string
	}
)

const (
	UserAgentPrefix = "fioup/"

	HeaderKeyTag    = "x-ats-tags"
	HeaderKeyApps   = "x-ats-dockerapps"
	HeaderKeyTarget = "x-ats-target"
)

func NewGatewayClient(cfg *config.Config, apps []string, targetID string) (*GatewayClient, error) {
	client, err := transport.CreateClient(cfg.TomlConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTPS HttpClient to talk to Device Gateway: %w", err)
	}
	headers := map[string]string{
		"user-agent": UserAgentPrefix + "/1.0.0", // TODO: figure out version and append here
		HeaderKeyTag: cfg.GetTag(),
	}
	if apps != nil {
		headers[HeaderKeyApps] = strings.Join(apps, ",")
	}
	if targetID != "" {
		headers[HeaderKeyTarget] = targetID
	}
	return &GatewayClient{
		BaseURL:    cfg.GetServerBaseURL(),
		HttpClient: client,
		Headers:    headers,
	}, nil
}

func (c *GatewayClient) Get(resourcePath string) (*transport.HttpRes, error) {
	return transport.HttpGet(c.HttpClient, c.BaseURL.JoinPath(resourcePath).String(), c.Headers)
}

func (c *GatewayClient) getJson(resourcePath string, item any) error {
	res, err := c.Get(resourcePath)
	if err != nil {
		return err
	}
	return res.Json(item)
}

func (c *GatewayClient) Post(resourcePath string, data any) (*transport.HttpRes, error) {
	return transport.HttpDo(c.HttpClient, http.MethodPost, c.BaseURL.JoinPath(resourcePath).String(), c.Headers, data)
}

func (c *GatewayClient) UpdateHeaders(apps []string, targetID string) {
	c.Headers[HeaderKeyApps] = strings.Join(apps, ",")
	c.Headers[HeaderKeyTarget] = targetID
}
