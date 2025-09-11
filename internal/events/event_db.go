// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package events

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/foundriesio/composeapp/pkg/compose"
	_ "github.com/foundriesio/composeapp/pkg/update"
	"github.com/rs/zerolog/log"
	_ "modernc.org/sqlite"
)

func CreateEventsTable(dbFilePath string) error {
	db, err := sql.Open("sqlite", dbFilePath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Err(closeErr).Msgf("failed to close database")
		}
	}()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS report_events(id INTEGER PRIMARY KEY, json_string TEXT NOT NULL);")
	if err != nil {
		return fmt.Errorf("failed to create report_events table: %w", err)
	}

	return nil
}

func SaveEvent(dbFilePath string, event *DgUpdateEvent) error {
	db, err := sql.Open("sqlite", dbFilePath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Err(closeErr).Msgf("failed to close database")
		}
	}()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event to JSON: %w", err)
	}

	_, err = db.Exec("INSERT INTO report_events (json_string) VALUES (?);", string(eventJSON))
	if err != nil {
		return fmt.Errorf("failed to insert event into report_events: %w", err)
	}

	return nil
}

func DeleteEvents(dbFilePath string, maxId int) error {
	db, err := sql.Open("sqlite", dbFilePath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Err(closeErr).Msgf("failed to close database")
		}
	}()

	_, err = db.Exec("DELETE FROM report_events WHERE id <= ?;", maxId)
	if err != nil {
		return fmt.Errorf("failed to delete event from report_events: %w", err)
	}

	return nil
}

func GetEvents(dbFilePath string) ([]DgUpdateEvent, int, error) {
	db, err := sql.Open("sqlite", dbFilePath)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Err(closeErr).Msgf("failed to close database")
		}
	}()

	rows, err := db.Query("SELECT id, json_string FROM report_events;")
	if err != nil {
		return nil, -1, fmt.Errorf("failed to select events: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			log.Err(closeErr).Msgf("failed to close rows")
		}
	}()

	maxId := -1
	var eventsList []DgUpdateEvent
	for rows.Next() {
		var eventData string
		var id int
		if err := rows.Scan(&id, &eventData); err != nil {
			return nil, -1, fmt.Errorf("failed to scan event data: %w", err)
		}

		var event DgUpdateEvent
		if err := json.Unmarshal([]byte(eventData), &event); err != nil {
			return nil, -1, fmt.Errorf("failed to unmarshal event data: %w", err)
		}

		if maxId < id {
			maxId = id
		}
		eventsList = append(eventsList, event)
	}

	if err := rows.Err(); err != nil {
		return nil, -1, fmt.Errorf("error iterating over rows: %w", err)
	}

	return eventsList, maxId, nil
}
