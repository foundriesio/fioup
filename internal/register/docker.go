// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package register

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

func getDockerConfigPath() (string, error) {
	sudoer := os.Getenv("SUDO_USER")
	path := ""
	if len(sudoer) > 0 {
		u, err := user.Lookup(sudoer)
		if err != nil {
			return "", fmt.Errorf("unable to configure docker-credential-helper: %w", err)
		}
		path = u.HomeDir
	} else {
		var err error
		path, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to configure docker-credential-helper: %w", err)
		}
	}
	path = filepath.Join(path, ".docker")

	if err := os.Mkdir(path, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return "", fmt.Errorf("unable to configure docker-credential-helper: %w", err)
	}

	return path, nil
}

func stageDockerChanges(opt *RegisterOptions) error {
	api := os.Getenv(ENV_DEVICE_API)
	hubUrl := "hub.foundries.io"
	if api != "" {
		uri, err := url.Parse(api)
		if err != nil {
			return fmt.Errorf("invalid DEVICE_API override: %w", err)
		}
		hubUrl = strings.Replace(uri.Hostname(), "api.", "hub.", 1)
	}
	slog.Debug("Factory registry",
		"uri", hubUrl)

	path, err := getDockerConfigPath()
	if err != nil {
		return err
	}
	slog.Debug("Docker config",
		"path", path)

	opt.dockerCfgPath = filepath.Join(path, "config.json")
	var config map[string]any
	if configBytes, err := os.ReadFile(opt.dockerCfgPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("unable to read docker configuration: %w", err)
	} else if err == nil {
		slog.Debug("Ammending existing docker config.json")
		if err = json.Unmarshal(configBytes, &config); err != nil {
			return fmt.Errorf("unable to update docker configuration for credential helper: %w", err)
		}
	} else {
		config = make(map[string]any)
	}

	helpers, ok := config["credHelpers"]
	if !ok {
		config["credHelpers"] = map[string]string{
			hubUrl: "fioup",
		}
	} else {
		helpers.(map[string]any)[hubUrl] = "fioup"
	}

	if configBytes, err := json.MarshalIndent(config, "", "  "); err != nil {
		return fmt.Errorf("unable to configure docker-credential-helper: %w", err)
	} else if err = writeSafely(opt.dockerCfgPath+".tmp", string(configBytes)); err != nil {
		return fmt.Errorf("unable to configure docker-credential-helper: %w", err)
	}

	return nil
}

func commitDockerChanges(opt *RegisterOptions) error {
	// configure the credential helper by symlinking this binary to docker-credential-fioup
	self, err := os.Readlink("/proc/self/exe")
	if err != nil {
		return fmt.Errorf("unable to configure docker-credential-helper. Can't find self: %w", err)
	}
	slog.Debug("fioup binary found",
		"self", self)

	if path, err := exec.LookPath("docker-credential-fioup"); err != nil {
		if os.Getegid() == 0 {
			if err := os.Symlink(self, "/usr/local/bin/docker-credential-fioup"); err != nil {
				return fmt.Errorf("unable to configure docker-credential-helper. Can't link to self: %w", err)
			}
		} else {
			paths := strings.Split(os.Getenv("PATH"), ":")
			slog.Debug("Looking for writable location",
				"paths", paths)
			found := false
			for _, path := range paths {
				dst := filepath.Join(path, "docker-credential-fioup")
				if err := os.Symlink(self, dst); err == nil {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("unable to configure docker-credential-helper. Can't find writeable location under user's PATH=%s", paths)
			}
		}
	} else {
		slog.Debug("Credential helper already installed",
			"path", path)
	}

	// complete transaction by setting the config file
	if err := os.Rename(opt.dockerCfgPath+".tmp", opt.dockerCfgPath); err != nil {
		return fmt.Errorf("unable to configure docker-credential-helper: %w", err)
	}
	return nil
}
