// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package events

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/foundriesio/fioup/pkg/fioup/client"
	"github.com/google/uuid"
)

type EventTypeValue string

const (
	DownloadStarted       EventTypeValue = "EcuDownloadStarted"
	DownloadCompleted     EventTypeValue = "EcuDownloadCompleted"
	InstallationStarted   EventTypeValue = "EcuInstallationStarted"
	InstallationApplied   EventTypeValue = "EcuInstallationApplied"
	InstallationCompleted EventTypeValue = "EcuInstallationCompleted"
)

type DgEvent struct {
	CorrelationId string `json:"correlationId"`
	Success       *bool  `json:"success"`
	TargetName    string `json:"targetName"`
	Version       string `json:"version"`
	Details       string `json:"details,omitempty"`
}
type DgEventType struct {
	Id      EventTypeValue `json:"id"`
	Version int            `json:"version"`
}
type DgUpdateEvent struct {
	Id         string      `json:"id"`
	DeviceTime string      `json:"deviceTime"`
	Event      DgEvent     `json:"event"`
	EventType  DgEventType `json:"eventType"`
}

func NewEvent(eventType EventTypeValue, details string, success *bool, correlationId string, targetName string, version int) []DgUpdateEvent {
	evt := []DgUpdateEvent{
		{
			Id:         uuid.New().String(),
			DeviceTime: time.Now().Format(time.RFC3339),
			Event: DgEvent{
				CorrelationId: correlationId,
				Success:       success,
				TargetName:    targetName,
				Version:       strconv.Itoa(version),
				Details:       details,
			},
			EventType: DgEventType{
				Id:      eventType,
				Version: 0,
			},
		},
	}
	return evt
}

func SendEvent(client *client.GatewayClient, event []DgUpdateEvent) error {
	res, err := client.Post("/events", event)
	if err != nil {
		slog.Error("Unable to send event", "error", err)
	} else if res.StatusCode < 200 || res.StatusCode > 204 {
		slog.Info(fmt.Sprintf("Server could not process event(%v): HTTP_%d - %s", interface{}(event), res.StatusCode, res.String()))
	}
	return err
}

func FlushEvents(dbFilePath string, client *client.GatewayClient) error {
	evts, maxId, err := GetEvents(dbFilePath)
	if err != nil {
		return fmt.Errorf("error getting events: %w", err)
	}

	if len(evts) == 0 {
		slog.Debug("No events to send")
		return nil
	}

	slog.Debug(fmt.Sprintf("Flushing %d events", len(evts)))
	err = SendEvent(client, evts)
	if err != nil {
		return fmt.Errorf("error sending events: %w", err)
	}

	err = DeleteEvents(dbFilePath, maxId)
	if err != nil {
		return fmt.Errorf("error deleting events: %w", err)
	}
	return nil
}
