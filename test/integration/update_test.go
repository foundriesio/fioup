package integration_tests

import (
	"context"
	"strconv"
	"testing"

	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/pkg/api"
	"github.com/foundriesio/fioup/pkg/client"
)

func TestUpdateSequence(t *testing.T) {
	client.DefaultHttpOperations = mockHttpOperations{}
	tempDir = t.TempDir()

	config = createMockConfig(t)

	ctx := context.Background()

	target1 := genNewTarget(t, 100, 2, 150, false)
	target2 := genNewTarget(t, 101, 3, 160, false)
	target3 := genNewTarget(t, 102, 1, 170, true)

	targets := []*Target{target1}
	saveTargetsJson(t, targets)
	testUpdateTo(t, ctx, target1, targets)

	targets = []*Target{target1, target2}
	saveTargetsJson(t, targets)
	testUpdateTo(t, ctx, target2, targets)

	targets = []*Target{target1, target2, target3}
	saveTargetsJson(t, targets)
	testUpdateTo(t, ctx, target3, targets)
}

func testUpdateTo(t *testing.T, ctx context.Context, target *Target, allTargets []*Target) {
	clearEvents()
	targets, currentStatus, err := api.Check(ctx, config, api.WithTUF(false))
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(targets) != len(allTargets) {
		t.Fatalf("Number of targets (%d) does not match expected (%d)", len(targets), len(allTargets))
	}
	originalTargetID := currentStatus.TargetID

	err = api.Fetch(ctx, config, -1, api.WithTUF(false))
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	successVal := true
	expectedEvents := []events.DgUpdateEvent{
		{
			EventType: events.DgEventType{
				Id: events.DownloadStarted,
			},
			Event: events.DgEvent{
				Version: strconv.Itoa(target.Version),
			},
		},
		{
			EventType: events.DgEventType{
				Id: events.DownloadCompleted,
			},
			Event: events.DgEvent{
				Version: strconv.Itoa(target.Version),
				Success: &successVal,
			},
		},
	}
	checkEvents(t, target, expectedEvents)
	checkStatus(t, ctx, config.ComposeConfig(), originalTargetID)

	err = api.Install(ctx, config)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	expectedEvents = []events.DgUpdateEvent{
		{
			EventType: events.DgEventType{
				Id: events.InstallationStarted,
			},
			Event: events.DgEvent{
				Version: strconv.Itoa(target.Version),
			},
		},
		{
			EventType: events.DgEventType{
				Id: events.InstallationApplied,
			},
			Event: events.DgEvent{
				Version: strconv.Itoa(target.Version),
			},
		},
	}
	checkEvents(t, target, expectedEvents)
	checkStatus(t, ctx, config.ComposeConfig(), originalTargetID)

	err = api.Start(context.Background(), config)
	if !target.Bad && err != nil {
		t.Fatalf("Start failed: %v", err)
	} else if target.Bad && err == nil {
		t.Fatalf("Start succeeded but was expected to fail")
	}
	successVal = !target.Bad
	expectedEvents = []events.DgUpdateEvent{
		{
			EventType: events.DgEventType{
				Id: events.InstallationCompleted,
			},
			Event: events.DgEvent{
				Version: strconv.Itoa(target.Version),
				Success: &successVal,
			},
		},
	}
	checkEvents(t, target, expectedEvents)
	if target.Bad {
		checkStatus(t, ctx, config.ComposeConfig(), originalTargetID)
	} else {
		checkStatus(t, ctx, config.ComposeConfig(), target.ID)
	}
}
