package targets

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/theupdateframework/go-tuf/v2/metadata"
	_ "modernc.org/sqlite"
)

type TargetCustom struct {
	Version string `json:"version"`
}

func BoolPointer(b bool) *bool {
	return &b
}

const (
	updateModeCurrent int = 1
	updateModePending int = 2
	updateModeFailed  int = 3
)

func RegisterInstallationStarted(dbFilePath string, target *metadata.TargetFiles, correlationId string) error {
	return saveInstalledVersions(dbFilePath, target, correlationId, updateModePending)
}

func RegisterInstallationSuceeded(dbFilePath string, target *metadata.TargetFiles, correlationId string) error {
	return saveInstalledVersions(dbFilePath, target, correlationId, updateModeCurrent)
}

func RegisterInstallationFailed(dbFilePath string, target *metadata.TargetFiles, correlationId string) error {
	return saveInstalledVersions(dbFilePath, target, correlationId, updateModeFailed)
}

func CreateTargetsTable(dbFilePath string) error {
	db, err := sql.Open("sqlite", dbFilePath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS installed_versions(
	id INTEGER PRIMARY KEY, 
	ecu_serial TEXT NOT NULL,
	sha256 TEXT NOT NULL,
	name TEXT NOT NULL,
	hashes TEXT NOT NULL,
	length INTEGER NOT NULL DEFAULT 0,
	correlation_id TEXT NOT NULL DEFAULT "",
	is_current INTEGER NOT NULL CHECK (is_current IN (0,1)) DEFAULT 0,
	is_pending INTEGER NOT NULL CHECK (is_pending IN (0,1)) DEFAULT 0,
	was_installed INTEGER NOT NULL CHECK (was_installed IN (0,1)) DEFAULT 0,
	custom_meta TEXT NOT NULL DEFAULT ""
);`)
	if err != nil {
		return fmt.Errorf("failed to create installed_versions table: %w", err)
	}

	return nil
}

func IsFailingTarget(dbFilePath string, name string) (bool, error) {
	db, err := sql.Open("sqlite", dbFilePath)
	if err != nil {
		return false, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM installed_versions WHERE name = ? AND was_installed = 0;", name)
	if err != nil {
		return false, fmt.Errorf("failed to select installed_versions: %w", err)
	}

	var count int
	for rows.Next() {
		count++
	}

	if count > 0 {
		return true, nil
	}

	return false, nil
}

func GetCurrentTarget(dbFilePath string) (*metadata.TargetFiles, error) {
	target := &metadata.TargetFiles{}
	target.Custom = &json.RawMessage{}
	target.Path = "Initial Target" // default value, if there is no target data in DB

	db, err := sql.Open("sqlite", dbFilePath)
	if err != nil {
		return target, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT name, custom_meta FROM installed_versions WHERE is_current = 1;")
	if err != nil {
		return target, err
	}

	var name string
	var customMeta string

	for rows.Next() {
		if err = rows.Scan(&name, &customMeta); err != nil {
			return target, err
		}
	}

	if err = rows.Err(); err != nil {
		return target, err
	}

	log.Debug().Msgf("Current target: %s", name)

	if name != "" {
		target.Path = name
		if err = json.Unmarshal([]byte(customMeta), target.Custom); err != nil {
			return target, fmt.Errorf("failed to unmarshal custom metadata: %v '%s'", err, customMeta)
		}
	}

	return target, nil
}

func saveInstalledVersions(dbFilePath string, target *metadata.TargetFiles, correlationId string, updateMode int) error {
	log.Debug().Msgf("Saving installed version: %s, correlation ID: %s, mode: %d", target.Path, correlationId, updateMode)
	db, err := sql.Open("sqlite", dbFilePath)
	if err != nil {
		return err
	}
	defer db.Close()

	var oldWasInstalled *bool = nil
	// var oldName string = ""
	rows, err := db.Query("SELECT name, was_installed FROM installed_versions ORDER BY id DESC LIMIT 1;")
	if err != nil {
		return fmt.Errorf("failed to select installed_versions: %w", err)
	}

	for rows.Next() {
		var name string
		var wasInstalled bool
		if err = rows.Scan(&name, &wasInstalled); err != nil {
			return fmt.Errorf("get name: %w", err)
		}

		if name == target.Path {
			log.Debug().Msg("DB: Target was installed before")
			oldWasInstalled = BoolPointer(wasInstalled)
			// oldName = name
		}
	}

	if updateMode == updateModeCurrent {
		// unset 'current' and 'pending' on all versions for this ecu
		_, err = db.Exec("UPDATE installed_versions SET is_current = 0, is_pending = 0")
		if err != nil {
			return fmt.Errorf("failed to update installed 1 versions: %w", err)
		}

	} else if updateMode == updateModePending {
		// unset 'pending' on all versions for this ecu
		_, err = db.Exec("UPDATE installed_versions SET is_pending = 0")
		if err != nil {
			return fmt.Errorf("failed to update installed 2 versions: %w", err)
		}
	}

	if oldWasInstalled != nil {
		if updateMode == updateModeFailed {
			_, err = db.Exec(
				"UPDATE installed_versions SET is_pending = 0, was_installed = 0 WHERE name = ?;",
				target.Path,
			)
			if err != nil {
				return fmt.Errorf("failed to save installed versions: %w", err)
			}
		} else {
			_, err = db.Exec(
				"UPDATE installed_versions SET correlation_id = ?, is_current = ?, is_pending = ?, was_installed = ? WHERE name = ?;",
				correlationId,
				updateMode == updateModeCurrent,                     // is_current
				updateMode == updateModePending,                     // is_pending
				updateMode == updateModeCurrent || *oldWasInstalled, // was_installed
				target.Path,
			)
			if err != nil {
				return fmt.Errorf("failed to save installed versions: %w", err)
			}
		}
	} else {
		customMeta, err := json.Marshal(target.Custom)
		if err != nil {
			return fmt.Errorf("failed to marshal custom metadata: %w", err)
		}
		sha256 := hex.EncodeToString(target.Hashes["sha256"])
		_, err = db.Exec(
			"INSERT INTO installed_versions (ecu_serial, sha256, name, hashes, length, custom_meta, correlation_id, is_current, is_pending, was_installed) VALUES (?,?,?,?,?,?,?,?,?,?);",
			"",
			sha256,
			target.Path,
			"sha256:"+sha256,
			target.Length,
			string(customMeta),
			correlationId,
			updateMode == updateModeCurrent, // is_current
			updateMode == updateModePending, // is_pending
			updateMode == updateModeCurrent, // was_installed
		)
		if err != nil {
			return fmt.Errorf("failed to save installed versions: %w", err)
		}
	}

	return nil
}
