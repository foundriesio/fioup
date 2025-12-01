package integration_tests

import (
	"testing"

	"github.com/foundriesio/fioup/pkg/api"
)

// TestResume tests invalid operations are rejected while resuming an update
// and that the update can be resumed correctly.
func TestResume(t *testing.T) {
	it := newIntegrationTest(t)

	target1 := it.genNewTarget(100, 2, 50, false)
	target2 := it.genNewTarget(101, 3, 60, false)
	target3 := it.genNewTarget(102, 1, 70, true)

	allTargets := []*Target{target1, target2}
	it.saveTargetsJson(allTargets)

	targets, _, err := api.Check(it.ctx, it.config, it.apiOpts...)

	checkErr(it.t, err)
	if len(targets) != len(allTargets) {
		t.Fatalf("Number of targets (%d) does not match expected (%d)", len(targets), len(allTargets))
	}

	// Start update to target1
	err = api.Fetch(it.ctx, it.config, target1.Version, it.apiOpts...)
	checkErr(it.t, err)

	// Calling fetch again should be a no-op and succeed
	err = api.Fetch(it.ctx, it.config, target1.Version, it.apiOpts...)
	checkErr(it.t, err)

	// Should not allow to perform a fetch to a different target while update is in progress
	err = api.Fetch(it.ctx, it.config, target2.Version, it.apiOpts...)
	if err == nil {
		t.Fatalf("Fetch is expected to fail but did not")
	}

	// Should keep updating to target1, even having target2 available
	err = api.Fetch(it.ctx, it.config, -1, it.apiOpts...)
	checkErr(it.t, err)

	allTargets = []*Target{target1, target2, target3}
	it.saveTargetsJson(allTargets)

	targets, _, err = api.Check(it.ctx, it.config, it.apiOpts...)
	checkErr(it.t, err)
	if len(targets) != len(allTargets) {
		t.Fatalf("Number of targets (%d) does not match expected (%d)", len(targets), len(allTargets))
	}

	// Should keep updating to target1, even with changes TUF
	err = api.Fetch(it.ctx, it.config, -1, it.apiOpts...)
	checkErr(it.t, err)

	// No install yet, start should fail
	err = api.Start(it.ctx, it.config, api.WithGatewayClient(it.gwClient))
	if err == nil {
		t.Fatalf("Start is expected to fail but did not")
	}

	// Install target1
	err = api.Install(it.ctx, it.config, api.WithGatewayClient(it.gwClient))
	checkErr(it.t, err)

	// Start target1
	err = api.Start(it.ctx, it.config, api.WithGatewayClient(it.gwClient))
	checkErr(it.t, err)

	// make sure we are now on target1
	it.checkStatus(target1.ID)
}
