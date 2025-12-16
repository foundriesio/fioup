// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package client

import (
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/foundriesio/fioconfig/transport"
	"github.com/foundriesio/fioup/pkg/config"
)

type (
	GwHttpOperations interface {
		HttpGet(client *http.Client, url string, headers map[string]string) (*transport.HttpRes, error)
		HttpDo(client *http.Client, method, url string, headers map[string]string, data any) (*transport.HttpRes, error)
	}

	GatewayClient struct {
		BaseURL    *url.URL
		HttpClient *http.Client
		Headers    map[string]string

		proxyAppsUrl string
		proxyAppsCa  *x509.CertPool

		lastNetInfoFile   string
		lastSotaFile      string
		sotaToReport      []byte
		lastHwinfoFile    string
		hwinfoToReport    []byte
		lastAppStatesFile string
		lastAppStates     map[string]AppState

		httpOperations GwHttpOperations
	}
)

const (
	UserAgentPrefix = "fioup/"

	HeaderKeyTag    = "x-ats-tags"
	HeaderKeyApps   = "x-ats-dockerapps"
	HeaderKeyTarget = "x-ats-target"
)

type transportHttpOperations struct{}

func (transportHttpOperations) HttpGet(client *http.Client, url string, headers map[string]string) (*transport.HttpRes, error) {
	return transport.HttpGet(client, url, headers)
}

func (transportHttpOperations) HttpDo(client *http.Client, method, url string, headers map[string]string, data any) (*transport.HttpRes, error) {
	return transport.HttpDo(client, method, url, headers, data)
}

type (
	GatewayClientOpts struct {
		HttpOperations GwHttpOperations
	}
	GatewayClientOpt func(*GatewayClientOpts)
)

func WithHttpOperations(ops GwHttpOperations) GatewayClientOpt {
	return func(o *GatewayClientOpts) {
		o.HttpOperations = ops
	}
}

func getGatewayClientOpts(options ...GatewayClientOpt) *GatewayClientOpts {
	opts := &GatewayClientOpts{transportHttpOperations{}}
	for _, o := range options {
		o(opts)
	}
	return opts
}

func NewGatewayClient(cfg *config.Config, apps []string, targetID string, options ...GatewayClientOpt) (*GatewayClient, error) {
	opts := getGatewayClientOpts(options...)
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

	var proxyCerts *x509.CertPool
	if len(cfg.GetComposeAppsProxy()) > 0 {
		proxyCerts = x509.NewCertPool()
		if pemData, err := os.ReadFile(cfg.GetComposeAppsProxyCA()); err != nil {
			return nil, fmt.Errorf("unable to read COMPOSE_APPS_PROXY_CA: %w", err)
		} else if ok := proxyCerts.AppendCertsFromPEM(pemData); !ok {
			return nil, fmt.Errorf("failed to set COMPOSE_APPS_PROXY_CA: %w", err)
		}
	}

	sota := cfg.GetStorageDir()
	gw := &GatewayClient{
		BaseURL:    cfg.GetServerBaseURL(),
		HttpClient: client,
		Headers:    headers,

		proxyAppsUrl: cfg.GetComposeAppsProxy(),
		proxyAppsCa:  proxyCerts,

		lastNetInfoFile:   filepath.Join(sota, ".last-netinfo"),
		lastSotaFile:      filepath.Join(sota, ".last-sota"),
		lastHwinfoFile:    filepath.Join(sota, ".last-hwinfo"),
		lastAppStatesFile: filepath.Join(sota, ".last-app-states"),

		httpOperations: opts.HttpOperations,
	}

	gw.initSota(cfg.TomlConfig())
	gw.initHwinfo()
	gw.initAppStateReporter()
	return gw, nil
}

func (c *GatewayClient) Get(resourcePath string) (*transport.HttpRes, error) {
	return c.httpOperations.HttpGet(c.HttpClient, c.BaseURL.JoinPath(resourcePath).String(), c.Headers)
}

func (c *GatewayClient) getJson(resourcePath string, item any) error {
	res, err := c.Get(resourcePath)
	if err != nil {
		return err
	}
	return res.Json(item)
}

func (c *GatewayClient) Post(resourcePath string, data any) (*transport.HttpRes, error) {
	return c.httpOperations.HttpDo(c.HttpClient, http.MethodPost, c.BaseURL.JoinPath(resourcePath).String(), c.Headers, data)
}

func (c *GatewayClient) Put(resourcePath string, data any) (*transport.HttpRes, error) {
	return c.httpOperations.HttpDo(c.HttpClient, http.MethodPut, c.BaseURL.JoinPath(resourcePath).String(), c.Headers, data)
}

func (c *GatewayClient) UpdateHeaders(apps []string, targetID string) {
	c.Headers[HeaderKeyApps] = strings.Join(apps, ",")
	c.Headers[HeaderKeyTarget] = targetID
}
