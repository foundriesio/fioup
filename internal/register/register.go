// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package register

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/rs/zerolog/log"
)

func sotaCleanup(opt *RegisterOptions) error {
	crt := opt.SotaDir + SOTA_PEM
	sql := opt.SotaDir + SOTA_SQL

	log.Info().Msg("Cleaning up SOTA files")

	if fileExists(sql) {
		log.Info().Msgf("Removing %s", sql)
		if err := os.Remove(sql); err != nil {
			log.Err(err).Msgf("unable to remove %s", sql)
			return err
		}
	}

	if fileExists(crt) {
		log.Info().Msgf("Removing %s", crt)
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
		log.Info().Msgf("ERROR: Device already registered in %s", opt.SotaDir)
		log.Info().Msg("Re-run with --force 1 to remove existing registration data")
		return fmt.Errorf("device already registered")
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

func putHSMInfo(opt *RegisterOptions, dev map[string]interface{}) {
	if opt.HsmModule == "" {
		return
	}
	dev["overrides.tls.pkey_source"] = "\"pkcs11\""
	dev["overrides.tls.cert_source"] = "\"pkcs11\""
	dev["overrides.storage.tls_pkey_path"] = ""
	dev["overrides.storage.tls_clientcert_path"] = ""
	dev["overrides.import.tls_pkey_path"] = ""
	dev["overrides.import.tls_clientcert_path"] = ""
}

func getDeviceInfo(opt *RegisterOptions, csr string, dev map[string]interface{}) {
	dev["use-ostree-server"] = strconv.FormatBool(opt.UseServer)
	dev["sota-config-dir"] = opt.SotaDir
	dev["hardware-id"] = opt.Hwid
	dev["name"] = opt.Name
	dev["uuid"] = opt.UUID
	dev["csr"] = csr

	putHSMInfo(opt, dev)
	// putComposeAppInfo(opt, dev) // Implement as needed

	if opt.DeviceGroup != "" {
		dev["group"] = opt.DeviceGroup
	}
	if opt.PacmanTags != "" {
		dev["overrides"] = map[string]interface{}{
			"pacman": map[string]interface{}{
				"tags": fmt.Sprintf("\"%s\"", opt.PacmanTags),
			},
		}
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

// We additionally write the entire p11 section. We can't tell the server the
// * PIN, and don't want to parse/modify TOML to add it, so just write the whole
// thing to /var/sota/
func fillP11EngineInfo(opt *RegisterOptions, toml *string) {
	*toml += "[p11]\n"
	*toml += fmt.Sprintf("module = \"%s\"\n", opt.HsmModule)
	*toml += fmt.Sprintf("pass = \"%s\"\n", opt.HsmPin)
	*toml += "tls_pkey_id = \"01\"\n"
	*toml += "tls_clientcert_id = \"03\"\n\n"
}

func populateSotaDir(opt *RegisterOptions, resp map[string]interface{}, pkey string) error {
	log.Info().Msg("Populate sota directory.")

	if opt.HsmModule == "" {
		// Write the private key
		if err := writeSafely(filepath.Join(opt.SotaDir, "pkey.pem"), pkey); err != nil {
			return err
		}
	}

	var sotaToml string
	for name, data := range resp {
		strData := fmt.Sprintf("%v", data)
		fullName := filepath.Join(opt.SotaDir, name)
		if filepath.Base(fullName) == "sota.toml" {
			sotaToml += strData + "\n"
			if opt.HsmModule != "" {
				fillP11EngineInfo(opt, &sotaToml)
			}
			continue
		}
		if err := writeSafely(fullName, strData); err != nil {
			goto errorHandler
		}
		if filepath.Ext(fullName) != ".pem" {
			continue
		}
		// Import the certificate to PKCS#11 if HSM is enabled
		if opt.HsmModule != "" {
			crt, err := readX509FromFile(fullName)
			if err != nil {
				goto errorHandler
			}
			if err := pkcs11StoreCert(opt, crt); err != nil {
				goto errorHandler
			}
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

func readX509FromFile(filename string) (*x509.Certificate, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM")
	}
	return x509.ParseCertificate(block.Bytes)
}

func pkcs11StoreCert(opt *RegisterOptions, crt *x509.Certificate) error {
	return nil
}

// Create a Certificate Signing Request
func createCSR(opt *RegisterOptions) (key string, csr string, err error) {
	if opt.HsmModule == "" {
		return OpenSSLCreateCSR(opt)
	}
	return pkcs11CreateCSR(opt)
}

func pkcs11CreateCSR(opt *RegisterOptions) (string, string, error) {
	return "", "", nil
}

// cleanup cleans up partial registration.
func cleanup(opt *RegisterOptions) {
	log.Info().Msg("Cleaning up partial registration before leaving")
	_ = sotaCleanup(opt)
	pkcs11Cleanup(opt)
}

func pkcs11Cleanup(opt *RegisterOptions) {
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

func RegisterDevice(opt *RegisterOptions) error {
	err := UpdateOptions(os.Args, opt)
	if err != nil {
		log.Err(err).Msg("Error parsing options")
		return err
	}

	// Check if this device can be registered
	if err := checkDeviceStatus(opt); err != nil {
		return err
	}

	headers, err := AuthGetHttpHeaders(opt)
	if err != nil {
		log.Err(err).Msg("Error getting HTTP headers")
		return err
	}

	// Check server reachability
	if err := AuthPingServer(); err != nil {
		return err
	}

	// Register signal handler for cleanup
	unsetSignals := setSignals(opt)
	defer unsetSignals()

	// Create the key pair and the certificate request
	key, csr, err := createCSR(opt)
	if err != nil {
		cleanup(opt)
		return err
	}

	// Get the device information
	info := make(map[string]interface{})
	getDeviceInfo(opt, csr, info)

	// Register the device with the factory
	log.Info().Msgf("Registering device %s with factory %s\n", opt.Name, opt.Factory)
	resp, err := AuthRegisterDevice(headers, info)
	if err != nil {
		cleanup(opt)
		return err
	}

	// Store the login details
	if err := populateSotaDir(opt, resp, key); err != nil {
		cleanup(opt)
		return err
	}

	log.Info().Msg("Device is now registered.")
	// if opt.StartDaemon {
	// 	fmt.Printf("Starting %s daemon\n", SOTA_CLIENT)
	// 	spawn("systemctl", "start", SOTA_CLIENT)
	// }
	return nil
}
