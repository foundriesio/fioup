package integration_tests

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/foundriesio/fioconfig/transport"
	"github.com/foundriesio/fioup/internal/events"
	cfg "github.com/foundriesio/fioup/pkg/config"
)

var (
	config *cfg.Config
)

type mockHttpOperations struct{}

func (mockHttpOperations) HttpGet(client *http.Client, url string, headers map[string]string) (*transport.HttpRes, error) {
	err := os.MkdirAll(tempDir+"/http_get", 0o700)
	if err != nil {
		return nil, fmt.Errorf("unable to create http_get dir: %w", err)
	}
	filePath := strings.Replace(url, config.GetServerBaseURL().String(), tempDir+"/http_get", 1)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read mock file %s: %w", filePath, err)
	}

	res := transport.HttpRes{
		StatusCode: 200,
		Body:       data,
		Header:     http.Header{},
	}

	// fmt.Print("HttpGet: " + url + " -> " + filePath + "\n")
	return &res, nil
}

var postedEvents []events.DgUpdateEvent

func (mockHttpOperations) HttpDo(client *http.Client, method, url string, headers map[string]string, data any) (*transport.HttpRes, error) {
	// fmt.Print("HttpDo " + method + " " + url + "\n")
	if method == http.MethodPost {
		if strings.HasSuffix(url, "/events") {
			postedEvents = append(postedEvents, data.([]events.DgUpdateEvent)...)
		}
	}

	res := transport.HttpRes{
		StatusCode: 200,
		Header:     http.Header{},
	}

	return &res, nil
}

func checkEvents(t *testing.T, target *Target, expectedEvents []events.DgUpdateEvent) {
	if len(postedEvents) != len(expectedEvents) {
		t.Fatalf("Number of posted events (%d) does not match expected (%d)", len(postedEvents), len(expectedEvents))
	}

	expectedEventsCopy := expectedEvents[:]
	for _, ev := range postedEvents {
		evtVersion, err := strconv.Atoi(ev.Event.Version)
		if err != nil {
			t.Fatalf("Event version is not an integer: %v", err)
		}

		if evtVersion != target.Version {
			t.Fatalf("Event version does not match target version")
		}

		eventFoundIndex := -1
		for i, expectedEv := range expectedEventsCopy {
			if ev.EventType.Id == expectedEv.EventType.Id {
				if expectedEv.Event.Success != nil {
					if ev.Event.Success == nil || *ev.Event.Success != *expectedEv.Event.Success {
						t.Fatalf("Event success does not match expected for event type %s", ev.EventType.Id)
					}
				}
				eventFoundIndex = i
				break
			}
		}
		if eventFoundIndex == -1 {
			t.Fatalf("Event type %s not found in expected events", ev.EventType.Id)
		} else {
			// Remove matched event from expectedEventsCopy
			expectedEventsCopy = append(expectedEventsCopy[:eventFoundIndex], expectedEventsCopy[eventFoundIndex+1:]...)
		}
	}

	// Double checking that no events are left unmatched
	if len(expectedEventsCopy) != 0 {
		t.Fatalf("Not all expected events were found. Missing: %+v", expectedEventsCopy)
	}

	clearEvents()
}

func clearEvents() {
	postedEvents = []events.DgUpdateEvent{}
}
