// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package register

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/rs/zerolog/log"
)

func sotaCleanup(opt *RegisterOptions) error {
	crt := opt.SotaDir + SOTA_PEM
	sql := opt.SotaDir + SOTA_SQL

	log.Debug().Str("directory", opt.SotaDir).Msg("Cleaning up SOTA files")

	if fileExists(sql) {
		log.Debug().Str("file", sql).Msg("Removing file")
		if err := os.Remove(sql); err != nil {
			return err
		}
	}

	if fileExists(crt) {
		log.Debug().Str("file", crt).Msg("Removing file")
		if err := os.Remove(crt); err != nil {
			log.Err(err).Msgf("unable to remove %s", crt)
			return err
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
		log.Err(err).Msgf("is %s running?", SOTA_CLIENT)
		return fmt.Errorf("unable to open update client lock file: %w", err)
	}

	defer func() {
		if closeErr := lock.Close(); closeErr != nil {
			log.Err(closeErr).Msgf("failed to close lock")
		}
	}()

	// Try to acquire a shared lock (non-blocking)
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_SH|syscall.LOCK_NB); err == nil {
		// Lock acquired, so aklite is not running
		errFlock := syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
		if errFlock != nil {
			log.Err(errFlock).Msgf("failed to unlock %s", aklock)
		}
		return nil
	} else {
		log.Err(err).Msgf("%s already running", SOTA_CLIENT)
		return fmt.Errorf("%s already running", SOTA_CLIENT)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func checkDeviceStatus(opt *RegisterOptions) error {
	tmp := opt.SotaDir + "/.tmp"

	// Check directory is writable
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Err(err).Msgf("Unable to write to %s", opt.SotaDir)
		return err
	}

	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			log.Err(closeErr).Msgf("failed to close temp file")
		}
	}()

	err = os.Remove(tmp)
	if err != nil {
		log.Err(err).Msgf("Unable to remove %s", tmp)
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
	dev["hardware-id"] = HARDWARE_ID
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
			log.Err(closeErr).Msgf("failed to close temp file")
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

func populateSotaDir(opt *RegisterOptions, resp map[string]interface{}) error {
	log.Debug().Msg("Populate sota directory.")

	var sotaToml string
	for name, data := range resp {
		strData := fmt.Sprintf("%v", data)
		fullName := filepath.Join(opt.SotaDir, name)
		if err := writeSafely(fullName, strData); err != nil {
			goto errorHandler
		}
	}
	if err := writeSafely(filepath.Join(opt.SotaDir, "sota.toml"), sotaToml); err != nil {
		goto errorHandler
	}
	return nil
errorHandler:
	_ = sotaCleanup(opt)
	return errors.New("failed to populate sota directory")
}

// cleanup cleans up partial registration.
func cleanup(opt *RegisterOptions) {
	log.Debug().Msg("Cleaning up partial registration before leaving")
	if err := sotaCleanup(opt); err != nil {
		log.Err(err).Msg("Unable to clean up")
	}
}

// signalHandler handles signals for cleanup.
func signalHandler(opt *RegisterOptions) func(os.Signal) {
	return func(sig os.Signal) {
		log.Info().Msgf("Handling %s signal\n", sig)
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
	_, csr, err := openSSLCreateCSR(opt)
	if err != nil {
		cleanup(opt)
		return err
	}

	// Get the device information
	info := make(map[string]interface{})
	getDeviceInfo(opt, csr, info)

	// Register the device with the factory
	log.Debug().
		Str("name", opt.Name).
		Str("factory", opt.Factory).
		Msg("Registering device")
	resp, err := authRegisterDevice(headers, info)
	if err != nil {
		cleanup(opt)
		return err
	}

	// Store the login details
	if err := populateSotaDir(opt, resp); err != nil {
		cleanup(opt)
		return err
	}

	// if opt.StartDaemon {
	// 	fmt.Printf("Starting %s daemon\n", SOTA_CLIENT)
	// 	spawn("systemctl", "start", SOTA_CLIENT)
	// }
	return nil
}
