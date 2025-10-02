// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package events

import (
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/foundriesio/fioup/pkg/client"
	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/target"
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

type EventSender struct {
	dbPath   string
	gwClient *client.GatewayClient

	ticker    *time.Ticker
	stopChan  chan struct{}
	flushChan chan struct{}

	wg sync.WaitGroup
}

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

func sendEvent(client *client.GatewayClient, event []DgUpdateEvent) error {
	res, err := client.Post("/events", event)
	if err != nil {
		slog.Error("Unable to send event", "error", err)
	} else if res.StatusCode < 200 || res.StatusCode > 204 {
		slog.Info("Server could not process event", "http_status", res.StatusCode, "response", res.String(), "event", event)
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
	err = sendEvent(client, evts)
	if err != nil {
		return fmt.Errorf("error sending events: %w", err)
	}

	err = DeleteEvents(dbFilePath, maxId)
	if err != nil {
		return fmt.Errorf("error deleting events: %w", err)
	}
	return nil
}

func NewEventSender(cfg *config.Config, gwClient *client.GatewayClient) (*EventSender, error) {
	eventSender := &EventSender{
		dbPath:   cfg.GetDBPath(),
		gwClient: gwClient,
	}

	return eventSender, nil
}

func (s *EventSender) Start() {
	slog.Debug("Starting events sender")
	if s.ticker != nil {
		return
	}
	s.stopChan = make(chan struct{}, 1)
	s.flushChan = make(chan struct{}, 1)
	s.wg.Add(1)
	s.ticker = time.NewTicker(time.Duration(time.Second * 10))
	go func(stopChan chan struct{}, flushChan chan struct{}) {
		defer s.ticker.Stop()
		defer s.wg.Done()
		for {
			select {
			case <-stopChan:
				return
			case <-flushChan:
				err := FlushEvents(s.dbPath, s.gwClient)
				if err != nil {
					slog.Error("Error flushing events", "error", err)
				}
			case <-s.ticker.C:
				err := FlushEvents(s.dbPath, s.gwClient)
				if err != nil {
					slog.Error("Error flushing events", "error", err)
				}
			}
		}
	}(s.stopChan, s.flushChan)
}

func (s *EventSender) Stop() {
	slog.Debug("Stopping events sender")
	if s.ticker == nil {
		return
	}
	s.flushChan <- struct{}{}
	s.stopChan <- struct{}{}
	s.wg.Wait()
	s.ticker = nil
	s.stopChan = nil
	s.flushChan = nil
	slog.Debug("Events sender stopped")
}

func (s *EventSender) EnqueueEvent(eventType EventTypeValue, updateID string, toTarget target.Target, success ...bool) error {
	var completionStatus *bool
	if len(success) > 0 {
		completionStatus = &success[0]
	}
	evt := NewEvent(eventType, "", completionStatus, updateID, toTarget.ID, toTarget.Version)
	err := SaveEvent(s.dbPath, &evt[0])
	if err != nil {
		return fmt.Errorf("error saving event: %w", err)
	}
	s.FlushEventsAsync()
	return nil
}

func (s *EventSender) FlushEventsAsync() {
	s.flushChan <- struct{}{}
}
