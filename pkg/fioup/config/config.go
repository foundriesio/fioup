// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package config

import (
	"github.com/foundriesio/composeapp/pkg/compose"
	v1 "github.com/foundriesio/composeapp/pkg/compose/v1"
	"github.com/foundriesio/fioconfig/sotatoml"
	"path"
	"strings"
)

type (
	Config struct {
		tomlConfig    *sotatoml.AppConfig
		composeConfig *compose.Config
	}
)

func NewConfig(tomlConfig *sotatoml.AppConfig) (*Config, error) {
	composeCfg, err := newComposeConfig(tomlConfig)
	if err != nil {
		return nil, err
	}
	return &Config{
		tomlConfig:    tomlConfig,
		composeConfig: composeCfg,
	}, nil
}

func (c *Config) TomlConfig() *sotatoml.AppConfig {
	return c.tomlConfig
}

func (c *Config) ComposeConfig() *compose.Config {
	return c.composeConfig
}

func (c *Config) GetDBPath() string {
	return path.Join(c.tomlConfig.GetDefault("storage.path", "/var/sota"),
		c.tomlConfig.GetDefault("storage.sqldb_path", "sql.db"))
}

func (c *Config) GetEnabledApps() []string {
	if c.tomlConfig.GetDefault("pacman.compose_apps", "-") != "-" {
		return strings.Split(c.tomlConfig.GetDefault("pacman.compose_apps", ""), ",")
	}
	return []string{}
}

func newComposeConfig(config *sotatoml.AppConfig) (*compose.Config, error) {
	cfg, err := v1.NewDefaultConfig(
		v1.WithStoreRoot(config.GetDefault("pacman.reset_apps_root", "/var/sota/reset-apps")),
		v1.WithComposeRoot(config.GetDefault("pacman.compose_apps_root", "/var/sota/compose-apps")),
		v1.WithUpdateDB(path.Join(config.GetDefault("storage.path", "/var/sota"), "updates.db")),
	)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
