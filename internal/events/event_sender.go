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

const (
	UpdateInitStarted     EventTypeValue = "UpdateInitStarted"
	UpdateInitCompleted   EventTypeValue = "UpdateInitCompleted"
	DownloadStarted       EventTypeValue = "EcuDownloadStarted"
	DownloadCompleted     EventTypeValue = "EcuDownloadCompleted"
	InstallationStarted   EventTypeValue = "EcuInstallationStarted"
	InstallationApplied   EventTypeValue = "EcuInstallationApplied"
	InstallationCompleted EventTypeValue = "EcuInstallationCompleted"
)

type (
	EventSender struct {
		dbPath   string
		gwClient *client.GatewayClient

		ticker    *time.Ticker
		stopChan  chan struct{}
		flushChan chan struct{}

		wg sync.WaitGroup
	}

	EventTypeValue string

	DgEvent struct {
		CorrelationId string `json:"correlationId"`
		Success       *bool  `json:"success"`
		TargetName    string `json:"targetName"`
		Version       string `json:"version"`
		Details       string `json:"details,omitempty"`
	}

	DgEventType struct {
		Id      EventTypeValue `json:"id"`
		Version int            `json:"version"`
	}

	DgUpdateEvent struct {
		Id         string      `json:"id"`
		DeviceTime string      `json:"deviceTime"`
		Event      DgEvent     `json:"event"`
		EventType  DgEventType `json:"eventType"`
	}

	EnqueueEventOptions struct {
		Success *bool
		Details string
	}
	EnqueueEventOption func(*EnqueueEventOptions)
)

func WithEventStatus(success bool) EnqueueEventOption {
	return func(opts *EnqueueEventOptions) {
		opts.Success = &success
	}
}

func WithEventDetails(details string) EnqueueEventOption {
	return func(opts *EnqueueEventOptions) {
		opts.Details = details
	}
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

func (s *EventSender) EnqueueEvent(eventType EventTypeValue, updateID string, toTarget target.Target, options ...EnqueueEventOption) error {
	opts := &EnqueueEventOptions{}
	for _, opt := range options {
		opt(opts)
	}
	err := SaveEvent(s.dbPath, &DgUpdateEvent{
		Id:         uuid.New().String(),
		DeviceTime: time.Now().Format(time.RFC3339),
		Event: DgEvent{
			CorrelationId: updateID,
			Success:       opts.Success,
			TargetName:    toTarget.ID,
			Version:       strconv.Itoa(toTarget.Version),
			Details:       opts.Details,
		},
		EventType: DgEventType{
			Id:      eventType,
			Version: 0x2, // Define event version as 0x2 to distinguish from events sent by older agents
		},
	})
	if err != nil {
		return fmt.Errorf("error saving event: %w", err)
	}
	s.FlushEventsAsync()
	return nil
}

func (s *EventSender) FlushEventsAsync() {
	if s.flushChan == nil {
		slog.Error("Requested events flush on stopped sender")
		return
	}
	s.flushChan <- struct{}{}
}
