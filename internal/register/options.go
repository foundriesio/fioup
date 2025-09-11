// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package register

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/docker/distribution/uuid"
	"github.com/rs/zerolog/log"
	ini "gopkg.in/ini.v1"
)

type RegisterOptions struct {
	Production bool
	// StartDaemon bool
	SotaDir     string
	DeviceGroup string
	Factory     string
	Hwid        string
	PacmanTag   string
	ApiToken    string
	HsmModule   string
	HsmPin      string
	HsmSoPin    string
	UUID        string

	Name           string
	ApiTokenHeader string
	Force          bool
}

const (
	LMP_OS_STR     = "/etc/os-release"
	OS_FACTORY_TAG = "LMP_FACTORY_TAG"
	OS_FACTORY     = "LMP_FACTORY"
	GIT_COMMIT     = "unknown"

	// Environment Variables
	ENV_DEVICE_FACTORY = "DEVICE_FACTORY"
	ENV_PRODUCTION     = "PRODUCTION"
	ENV_OAUTH_BASE     = "OAUTH_BASE"
	ENV_DEVICE_API     = "DEVICE_API"

	// HSM definitions
	// HSM_TOKEN_STR   = "aktualizr"
	// HSM_TLS_STR     = "tls"
	// HSM_TLS_ID_STR  = "01"
	// HSM_CRT_STR     = "client"
	// HSM_CRT_ID      = 3
	// HSM_CRT_ID_STR  = "03"

	// Files
	AKLITE_LOCK = "/var/lock/aklite.lock"
	SOTA_DIR    = "/var/sota"
	SOTA_PEM    = "/client.pem"
	SOTA_SQL    = "/sql.db"

	SOTA_CLIENT = "aktualizr-lite"
)

func getFactoryTagsInfo(osRelease string) (factory, fsrc, tag, tsrc string) {
	if env := os.Getenv(ENV_DEVICE_FACTORY); env != "" {
		factory = env
		fsrc = "environment"
	}
	if _, err := os.Stat(osRelease); err != nil {
		return
	}
	cfg, err := ini.Load(osRelease)
	if err != nil {
		log.Info().Msgf("Can't parse file %s", osRelease)
		return
	}
	tag = cfg.Section("").Key(OS_FACTORY_TAG).String()
	tag = strings.ReplaceAll(tag, "\"", "")
	if tag != "" {
		tsrc = osRelease
	}
	if factory != "" {
		return
	}
	factory = cfg.Section("").Key(OS_FACTORY).String()
	factory = strings.ReplaceAll(factory, "\"", "")
	if factory != "" {
		fsrc = osRelease
	}
	return
}

func validateUUID(opt *RegisterOptions) error {
	_, err := uuid.Parse(opt.UUID)
	if err == nil {
		return nil
	}
	return fmt.Errorf("invalid UUID: %s", opt.UUID)
}

// func validateHSM(opt *LmpOptions) error {
// 	if opt.HsmModule == "" {
// 		if opt.HsmSoPin != "" || opt.HsmPin != "" {
// 			return errors.New("HSM incorrectly configured")
// 		}
// 		return nil
// 	}
// 	if opt.HsmSoPin == "" || opt.HsmPin == "" /* || pkcs11CheckHSM(opt) */ {
// 		return errors.New("HSM incorrectly configured")
// 	}
// 	return nil
// }

func getUUID(opt *RegisterOptions) error {
	if opt.UUID != "" {
		return validateUUID(opt)
	}
	if opt.UUID == "" {
		opt.UUID = uuid.Generate().String()
		log.Info().Msgf("UUID: %s [Random]", opt.UUID)
	}
	return validateUUID(opt)
}

func UpdateOptions(args []string, opt *RegisterOptions) error {
	factory, fsrc, tag, tsrc := getFactoryTagsInfo(LMP_OS_STR)
	if opt.Factory == "" || opt.Factory == "lmp" {
		return errors.New("missing factory definition")
	}
	if opt.PacmanTag == "" {
		return errors.New("missing tag definition")
	}
	if factory != opt.Factory {
		log.Info().Msg("Factory read from command line")
	} else {
		log.Info().Msgf("Factory read from %s", fsrc)
	}
	if tag != opt.PacmanTag {
		log.Info().Msg("Tag read from command line")
	} else {
		log.Info().Msgf("Tag read from %s", tsrc)
	}
	// if err := validateHSM(opt); err != nil {
	// 	return err
	// }
	if os.Getenv(ENV_PRODUCTION) != "" {
		opt.Production = true
	}
	if err := getUUID(opt); err != nil {
		return err
	}
	if opt.Name == "" {
		log.Info().Msg("Setting device name to UUID")
		opt.Name = opt.UUID
	}
	return nil
}
