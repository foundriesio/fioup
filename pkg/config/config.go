// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package config

import (
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/foundriesio/composeapp/pkg/compose"
	v1 "github.com/foundriesio/composeapp/pkg/compose/v1"
	"github.com/foundriesio/fioconfig/sotatoml"
	"github.com/foundriesio/fioconfig/transport"
)

type (
	Config struct {
		tomlConfig       *sotatoml.AppConfig
		composeConfig    *compose.Config
		dgBaseURL        *url.URL
		storageWatermark uint
		proxyProvider    *ProxyProvider
	}
	// HttpClientFunc defines a function type for making HTTP requests via a proxy.
	// It takes the method, URL, headers, and body as parameters and returns an HttpRes and an error.
	proxyHTTPClient func(string, string, map[string]string, any) (*transport.HttpRes, error)
	ProxyProvider   struct {
		client           proxyHTTPClient
		proxyUrlProvider string
		proxyCerts       *x509.CertPool
	}
)

const (
	TagKey                = "pacman.tags"
	ServerBaseUrlKey      = "tls.server"
	StorageDirKey         = "storage.path"
	HardwareIDKey         = "provision.primary_ecu_hardware_id"
	StorageUsageWatermark = "pacman.storage_watermark" // in percentage of overall storage, the maximum allowed to be used by apps
	ComposeAppsProxyKey   = "pacman.compose_apps_proxy"
	ComposeAppsProxyCaKey = "import.tls_cacert_path"

	StorageDefaultDir               = "/var/sota"
	TargetsDefaultFilename          = "targets.json"
	StorageUsageWatermarkDefaultStr = "95"
	StorageUsageWatermarkDefault    = 95
	MinStorageUsageWatermark        = 20
	MaxStorageUsageWatermark        = 99
)

func NewConfig(tomlConfigPaths []string) (*Config, error) {
	var err error
	cfg := &Config{}

	if len(tomlConfigPaths) == 0 {
		return nil, fmt.Errorf("config: no TOML paths provided")
	}
	if cfg.tomlConfig, err = sotatoml.NewAppConfig(tomlConfigPaths); err != nil {
		return nil, fmt.Errorf("config: failed to load TOML from paths %q: %w",
			strings.Join(tomlConfigPaths, ", "), err)
	}
	// Check mandatory fields in the TOML config
	if !cfg.tomlConfig.Has(ServerBaseUrlKey) {
		return nil, fmt.Errorf("no %q is found in the TOML config;"+
			" it defines the device gateway base URL", ServerBaseUrlKey)
	}
	cfg.dgBaseURL, err = url.Parse(cfg.tomlConfig.Get(ServerBaseUrlKey))
	if err != nil {
		return nil, fmt.Errorf("invalid value of the device gateway base URL: %w", err)
	}
	// Validate and set storage usage watermark
	cfg.storageWatermark = StorageUsageWatermarkDefault
	watermarkStr := cfg.tomlConfig.GetDefault(StorageUsageWatermark, StorageUsageWatermarkDefaultStr)
	if watermark, err := strconv.Atoi(watermarkStr); err == nil {
		if watermark < MinStorageUsageWatermark || watermark > MaxStorageUsageWatermark {
			slog.Warn("storage usage watermark out of range; using default", "value", watermark, "default", StorageUsageWatermarkDefaultStr)
		} else {
			cfg.storageWatermark = uint(watermark)
		}
	} else {
		slog.Warn("invalid storage usage watermark value; using default", "value", watermarkStr, "default", StorageUsageWatermarkDefaultStr)
	}
	slog.Debug("storage usage watermark set", "value", cfg.storageWatermark)

	var composeProxyProvider compose.ProxyProvider
	if cfg.tomlConfig.Has(ComposeAppsProxyKey) && cfg.tomlConfig.Get(ComposeAppsProxyKey) != "" {
		if p, err := newProxyProvider(cfg.tomlConfig); err == nil {
			cfg.proxyProvider = p
			composeProxyProvider = cfg.proxyProvider.getComposeAppProxy()
			slog.Debug("proxy for compose apps is set", "proxy URL", p.proxyUrlProvider)
		} else {
			slog.Error("failed to create proxy provider for compose apps; skip using proxy", "error", err)
		}
	}
	if cfg.composeConfig, err = newComposeConfig(cfg.tomlConfig, composeProxyProvider); err != nil {
		return nil, fmt.Errorf("failed to create compose config: %w", err)
	}

	return cfg, nil
}

func (c *Config) GetHardwareID() string {
	return c.tomlConfig.Get(HardwareIDKey)
}

func (c *Config) GetTargetsFilepath() string {
	return filepath.Join(c.GetStorageDir(), TargetsDefaultFilename)
}

func (c *Config) GetStorageDir() string {
	return c.tomlConfig.GetDefault(StorageDirKey, StorageDefaultDir)
}

func (c *Config) GetTag() string {
	return c.tomlConfig.Get(TagKey)
}

func (c *Config) GetServerBaseURL() *url.URL {
	return c.dgBaseURL
}

func (c *Config) TomlConfig() *sotatoml.AppConfig {
	return c.tomlConfig
}

func (c *Config) ComposeConfig() *compose.Config {
	return c.composeConfig
}

func (c *Config) GetDBPath() string {
	// TODO: set the defaults in cmd/fioup package instead of here
	return filepath.Join(c.tomlConfig.GetDefault("storage.path", "/var/sota"),
		c.tomlConfig.GetDefault("storage.sqldb_path", "sql.db"))
}

func (c *Config) GetEnabledApps() []string {
	if !c.tomlConfig.Has("pacman.compose_apps") {
		// If "compose_apps" is not set then return nil to indicate all apps are enabled
		// (vs. an empty list which would mean no apps are enabled)
		return nil
	}
	apps := c.tomlConfig.Get("pacman.compose_apps")
	parts := strings.Split(apps, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			result = append(result, v)
		}
	}
	return result
}

func (c *Config) GetStorageUsageWatermark() uint {
	return c.storageWatermark
}

func (c *Config) SetClientForProxy(client proxyHTTPClient) {
	if c.proxyProvider != nil {
		c.proxyProvider.client = client
	}
}

func newComposeConfig(config *sotatoml.AppConfig, proxyProvider compose.ProxyProvider) (*compose.Config, error) {
	// TODO: set the defaults in cmd/fioup package instead of here
	return v1.NewDefaultConfig(
		v1.WithStoreRoot(config.GetDefault("pacman.reset_apps_root", "/var/sota/reset-apps")),
		v1.WithComposeRoot(config.GetDefault("pacman.compose_apps_root", "/var/sota/compose-apps")),
		v1.WithUpdateDB(path.Join(config.GetDefault("storage.path", "/var/sota"), "updates.db")),
		v1.WithProxy(proxyProvider),
	)
}

func newProxyProvider(cfg *sotatoml.AppConfig) (*ProxyProvider, error) {
	var proxyCerts *x509.CertPool

	proxyCa := cfg.Get(ComposeAppsProxyCaKey)
	if len(proxyCa) == 0 {
		return nil, fmt.Errorf("path to proxy CA certificates not set in config key %s ", ComposeAppsProxyCaKey)
	}
	proxyCerts = x509.NewCertPool()
	if b, err := os.ReadFile(proxyCa); err == nil {
		if ok := proxyCerts.AppendCertsFromPEM(b); !ok {
			return nil, fmt.Errorf("failed to parse proxy CA certificates from %s", proxyCa)
		}
	} else {
		return nil, fmt.Errorf("failed to read proxy CA certificates from %s: %w", proxyCa, err)
	}

	return &ProxyProvider{
		proxyUrlProvider: cfg.Get(ComposeAppsProxyKey),
		proxyCerts:       proxyCerts,
	}, nil
}

func (p *ProxyProvider) getComposeAppProxy() compose.ProxyProvider {
	return func() *compose.ProxyConfig {
		if p.client == nil {
			slog.Error("gateway client is not initialized; skip using proxy")
			return nil
		}
		resp, err := p.client(http.MethodPost, p.proxyUrlProvider, nil, nil)
		if err != nil {
			slog.Error("failed to request apps proxy URL; skip using proxy", "error", err)
			return nil
		} else if resp.StatusCode != 201 {
			slog.Error("unexpected response code when requesting apps proxy URL; skip using proxy", "code", resp.StatusCode)
			return nil
		}
		var proxyURL *url.URL
		if proxyURL, err = url.ParseRequestURI(resp.String()); err != nil {
			slog.Error("invalid proxy URL received from server; skip using proxy", "url", resp.String(), "error", err)
			return nil
		}
		if proxyURL.Scheme != "http" && proxyURL.Scheme != "https" {
			slog.Error("unsupported proxy URL scheme; skip using proxy", "url", resp.String())
			return nil
		}
		if proxyURL.Host == "" {
			slog.Error("empty proxy URL host; skip using proxy", "url", resp.String())
			return nil
		}
		return &compose.ProxyConfig{
			ProxyURL:   proxyURL,
			ProxyCerts: p.proxyCerts,
		}
	}
}
