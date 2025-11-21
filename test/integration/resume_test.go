package integration_tests

import (
	"context"
	"testing"

	"github.com/foundriesio/fioup/pkg/api"
	"github.com/foundriesio/fioup/pkg/client"
)

func TestResume(t *testing.T) {
	client.DefaultHttpOperations = mockHttpOperations{}
	tempDir = t.TempDir()

	config = createMockConfig(t)
	ctx := context.Background()

	target1 := genNewTarget(t, 100, 2, 50, false)
	target2 := genNewTarget(t, 101, 3, 60, false)
	target3 := genNewTarget(t, 102, 1, 70, true)

	allTargets := []*Target{target1, target2}
	saveTargetsJson(t, allTargets)

	targets, _, err := api.Check(ctx, config, api.WithTUF(false))

	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(targets) != len(allTargets) {
		t.Fatalf("Number of targets (%d) does not match expected (%d)", len(targets), len(allTargets))
	}

	// Start update to target1
	err = api.Fetch(ctx, config, target1.Version, api.WithTUF(false))
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// Calling fetch again should be a no-op and succeed
	err = api.Fetch(ctx, config, target1.Version, api.WithTUF(false))
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// Should not allow to perform a fetch to a different target while update is in progress
	err = api.Fetch(ctx, config, target2.Version, api.WithTUF(false))
	if err == nil {
		t.Fatalf("Fetch is expected to fail but did not")
	}

	// Should keep updating to target1, even having target2 available
	err = api.Fetch(ctx, config, -1, api.WithTUF(false))
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	allTargets = []*Target{target1, target2, target3}
	saveTargetsJson(t, allTargets)

	targets, _, err = api.Check(ctx, config, api.WithTUF(false))
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(targets) != len(allTargets) {
		t.Fatalf("Number of targets (%d) does not match expected (%d)", len(targets), len(allTargets))
	}

	// Should keep updating to target1, even with changes TUF
	err = api.Fetch(ctx, config, -1, api.WithTUF(false))
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// No install yet, start should fail
	err = api.Start(ctx, config)
	if err == nil {
		t.Fatalf("Start is expected to fail but did not")
	}

	// Install target1
	err = api.Install(ctx, config)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Start target1
	err = api.Start(ctx, config)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// make sure we are now on target1
	checkStatus(t, ctx, config.ComposeConfig(), target1.ID)
}
