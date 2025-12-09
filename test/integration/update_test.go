package integration_tests

import (
	"context"
	"strconv"
	"testing"

	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/pkg/api"
)

// TestUpdateSequence tests a sequence of updates including good and bad targets, verifying
// that events are posted correctly and status is updated as expected.
func TestUpdateSequence(t *testing.T) {
	it := newIntegrationTest(t)

	target1 := it.genNewTarget(100, 2, 50, false, "")
	target2 := it.genNewTarget(101, 3, 60, false, "")
	target3 := it.genNewTarget(102, 1, 70, true, "")

	targets := []*Target{target1}
	it.saveTargetsJson(targets)
	it.testUpdateTo(target1, targets)

	targets = []*Target{target1, target2}
	it.saveTargetsJson(targets)
	it.testUpdateTo(target2, targets)

	targets = []*Target{target1, target2, target3}
	it.saveTargetsJson(targets)
	it.testUpdateTo(target3, targets)

	// Sync to target2
	targets = []*Target{target1, target2}
	it.saveTargetsJson(targets)
	it.testUpdateTo(target2, targets)

	// Downgrade to target1
	targets = []*Target{target1}
	it.saveTargetsJson(targets)
	it.testUpdateTo(target1, targets)
}

func (it *integrationTest) testUpdateTo(target *Target, allTargets []*Target) {
	it.t.Helper()
	clearEvents()
	targets, currentStatus, err := api.Check(it.ctx, it.config, it.apiOpts...)
	beforeApps := runningAppsURIs(currentStatus)
	checkErr(it.t, err)
	if len(targets) != len(allTargets) {
		it.t.Fatalf("Number of targets (%d) does not match expected (%d)", len(targets), len(allTargets))
	}
	originalTargetID := currentStatus.TargetID

	err = api.Fetch(it.ctx, it.config, -1, it.apiOpts...)
	checkErr(it.t, err)
	successVal := true
	expectedEvents := []events.DgUpdateEvent{
		{
			EventType: events.DgEventType{
				Id: events.UpdateInitStarted,
			},
			Event: events.DgEvent{
				Version: strconv.Itoa(target.Version),
			},
		},
		{
			EventType: events.DgEventType{
				Id: events.UpdateInitCompleted,
			},
			Event: events.DgEvent{
				Version: strconv.Itoa(target.Version),
				Success: &successVal,
			},
		},
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
	it.checkEvents(target, expectedEvents)
	it.checkStatus(originalTargetID, beforeApps, false)

	err = api.Install(it.ctx, it.config, it.apiOpts...)
	checkErr(it.t, err)
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
	it.checkEvents(target, expectedEvents)
	it.checkStatus(originalTargetID, []string{}, false)

	err = api.Start(context.Background(), it.config, it.apiOpts...)
	if target.Bad {
		if err == nil {
			it.t.Fatalf("Start succeeded but was expected to fail")
		}
	} else {
		checkErr(it.t, err)
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
	it.checkEvents(target, expectedEvents)
	if target.Bad {
		it.checkStatus(originalTargetID, []string{}, false)
	} else {
		it.checkStatus(target.ID, target.appsURIs(), true)
	}
}
