// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package config

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/foundriesio/composeapp/pkg/compose"
	v1 "github.com/foundriesio/composeapp/pkg/compose/v1"
	"github.com/foundriesio/fioconfig/sotatoml"
)

type (
	Config struct {
		tomlConfig    *sotatoml.AppConfig
		composeConfig *compose.Config
	}
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
	if cfg.composeConfig, err = newComposeConfig(cfg.tomlConfig); err != nil {
		return nil, fmt.Errorf("failed to create compose config: %w", err)
	}

	return cfg, nil
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

func newComposeConfig(config *sotatoml.AppConfig) (*compose.Config, error) {
	// TODO: set the defaults in cmd/fioup package instead of here
	return v1.NewDefaultConfig(
		v1.WithStoreRoot(config.GetDefault("pacman.reset_apps_root", "/var/sota/reset-apps")),
		v1.WithComposeRoot(config.GetDefault("pacman.compose_apps_root", "/var/sota/compose-apps")),
		v1.WithUpdateDB(path.Join(config.GetDefault("storage.path", "/var/sota"), "updates.db")),
	)
}
