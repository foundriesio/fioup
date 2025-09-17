// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package register

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

func sotaCleanup(opt *RegisterOptions) error {
	crt := opt.SotaDir + SOTA_PEM
	sql := opt.SotaDir + SOTA_SQL

	slog.Debug("Cleaning up SOTA files",
		"directory", opt.SotaDir)

	if fileExists(sql) {
		slog.Debug("Removing file",
			"file", sql)
		if err := os.Remove(sql); err != nil {
			return err
		}
	}

	if fileExists(crt) {
		slog.Debug("Removing file",
			"file", crt)
		if err := os.Remove(crt); err != nil {
			return fmt.Errorf("unable to remove %s: %w", crt, err)
		}
	}

	return nil
}

func checkSotaFiles(opt *RegisterOptions) error {
	crt := opt.SotaDir + SOTA_PEM
	sql := opt.SotaDir + SOTA_SQL

	crtMissing := !fileExists(crt)
	sqlMissing := !fileExists(sql)

	if crtMissing && sqlMissing {
		return nil
	}

	if !opt.Force {
		return os.ErrExist
	}

	return sotaCleanup(opt)
}

func checkUpdateClientNotRunning() error {
	// TODO: use lock in new update client?
	aklock := AKLITE_LOCK

	if !fileExists(aklock) {
		return nil
	}

	lock, err := os.OpenFile(aklock, os.O_RDONLY, 0600)
	if err != nil {
		return fmt.Errorf("unable to open update client lock file: %w", err)
	}

	defer func() {
		if closeErr := lock.Close(); closeErr != nil {
			slog.Error("failed to close lock", "error", closeErr)
		}
	}()

	// Try to acquire a shared lock (non-blocking)
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_SH|syscall.LOCK_NB); err == nil {
		// Lock acquired, so aklite is not running
		errFlock := syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
		if errFlock != nil {
			slog.Error("failed to unlock lock file", "lock_file", aklock, "error", errFlock)
		}
		return nil
	} else {
		return fmt.Errorf("%s already running: %w", SOTA_CLIENT, err)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func checkAndCreateSotaDir(sotaDir string) error {
	info, err := os.Stat(sotaDir)
	if os.IsNotExist(err) {
		slog.Debug("creating sota directory", "path", sotaDir)
		err = os.MkdirAll(sotaDir, 0700)
		if err != nil {
			return fmt.Errorf("unable to create %s: %w", sotaDir, err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("unable to access %s: %w", sotaDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", sotaDir)
	}
	return nil
}

func checkDeviceStatus(opt *RegisterOptions) error {
	tmp := opt.SotaDir + "/.tmp"

	// Check directory is writable
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to write to %s: %w", opt.SotaDir, err)
	}

	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			slog.Error("failed to close temp file", "error", closeErr)
		}
	}()

	err = os.Remove(tmp)
	if err != nil {
		slog.Error("Unable to remove temp file", "path", tmp, "error", err)
	}

	// Update client must not be running
	if err := checkUpdateClientNotRunning(); err != nil {
		return err
	}

	// Check device was not registered
	if err := checkSotaFiles(opt); err != nil {
		return err
	}

	return nil
}

func getDeviceInfo(opt *RegisterOptions, csr string, dev map[string]interface{}) {
	dev["use-ostree-server"] = "true"
	dev["sota-config-dir"] = opt.SotaDir
	dev["hardware-id"] = opt.HardwareID
	dev["name"] = opt.Name
	dev["uuid"] = opt.UUID
	dev["csr"] = csr

	dev["overrides"] = map[string]any{
		"pacman": map[string]any{
			"type":              "\"ostree+compose_apps\"",
			"reset_apps_root":   "\"" + filepath.Join(opt.SotaDir, "reset-apps") + "\"",
			"compose_apps_root": "\"" + filepath.Join(opt.SotaDir, "compose-apps") + "\"",
			"tags":              "\"" + opt.PacmanTag + "\"",
		},
	}

	// putComposeAppInfo(opt, dev) // Implement as needed

	if opt.DeviceGroup != "" {
		dev["group"] = opt.DeviceGroup
	}
}

func writeSafely(name, content string) error {
	tmp := name + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to open %s for writing: %w", tmp, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			slog.Error("failed to close temp file", "error", closeErr)
		}
	}()
	if _, err := io.WriteString(f, content); err != nil {
		return fmt.Errorf("unable to write to %s: %w", tmp, err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("unable to fsync %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, name); err != nil {
		return fmt.Errorf("unable to create %s: %w", name, err)
	}
	return nil
}

func populateSotaDir(opt *RegisterOptions, resp map[string]interface{}, pkey string) error {
	slog.Debug("Populate sota directory.")

	if err := writeSafely(filepath.Join(opt.SotaDir, "pkey.pem"), pkey); err != nil {
		return err
	}

	for name, data := range resp {
		strData := fmt.Sprintf("%v", data)
		fullName := filepath.Join(opt.SotaDir, name)
		if err := writeSafely(fullName, strData); err != nil {
			goto errorHandler
		}
	}
	return nil
errorHandler:
	_ = sotaCleanup(opt)
	return errors.New("failed to populate sota directory")
}

// cleanup cleans up partial registration.
func cleanup(opt *RegisterOptions) {
	slog.Debug("Cleaning up partial registration before leaving")
	if err := sotaCleanup(opt); err != nil {
		slog.Error("Unable to clean up", "error", err)
	}
}

// signalHandler handles signals for cleanup.
func signalHandler(opt *RegisterOptions) func(os.Signal) {
	return func(sig os.Signal) {
		slog.Info("Handling signal", "signal", sig)
		cleanup(opt)
		os.Exit(1)
	}
}

// setSignals sets up signal handlers for cleanup.
func setSignals(opt *RegisterOptions) func() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGSEGV)
	done := make(chan struct{})
	go func() {
		sig := <-sigs
		signalHandler(opt)(sig)
		close(done)
	}()
	return func() { signal.Stop(sigs); close(done) }
}

func RegisterDevice(opt *RegisterOptions, cb OauthCallback) error {
	err := updateOptions(opt)
	if err != nil {
		return err
	}
	// Check and create sota directory if needed and possible
	if err := checkAndCreateSotaDir(opt.SotaDir); err != nil {
		return err
	}
	// Check if this device can be registered
	if err := checkDeviceStatus(opt); err != nil {
		return err
	}

	headers, err := authGetHttpHeaders(opt, cb)
	if err != nil {
		return err
	}

	// Check server reachability
	if err := authPingServer(); err != nil {
		return err
	}

	// Register signal handler for cleanup
	unsetSignals := setSignals(opt)
	defer unsetSignals()

	// Create the key pair and the certificate request
	key, csr, err := openSSLCreateCSR(opt)
	if err != nil {
		cleanup(opt)
		return err
	}

	// Get the device information
	info := make(map[string]interface{})
	getDeviceInfo(opt, csr, info)

	if err := stageDockerChanges(opt); err != nil {
		cleanup(opt)
		return err
	}

	// Register the device with the factory
	slog.Debug("Registering device",
		"name", opt.Name,
		"factory", opt.Factory)
	resp, err := authRegisterDevice(headers, info)
	if err != nil {
		cleanup(opt)
		return err
	}

	// Store the login details
	if err := populateSotaDir(opt, resp, key); err != nil {
		cleanup(opt)
		return err
	}

	if err := commitDockerChanges(opt); err != nil {
		return err
	}

	// if opt.StartDaemon {
	// 	fmt.Printf("Starting %s daemon\n", SOTA_CLIENT)
	// 	spawn("systemctl", "start", SOTA_CLIENT)
	// }
	return nil
}
