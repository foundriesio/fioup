package integration_tests

import (
	"testing"

	"github.com/foundriesio/fioup/pkg/api"
	"github.com/foundriesio/fioup/pkg/state"
)

// Verify that the right error is returned in specific situations
func TestErrors(t *testing.T) {
	it := newIntegrationTest(t)
	it.setApps(nil)

	targets := []*Target{}
	it.saveTargetsJson(targets)

	_, _, err := api.Check(it.ctx, it.config, it.apiOpts...)
	expectErr(it.t, err, state.ErrTargetNotFound)

	err = api.Update(it.ctx, it.config, -1, it.apiOpts...)
	expectErr(it.t, err, state.ErrTargetNotFound)

	target1 := it.genNewTarget(100, 2, 50, false, "")

	targets = []*Target{target1}
	it.saveTargetsJson(targets)

	_, _, err = api.Check(it.ctx, it.config, it.apiOpts...)
	checkErr(it.t, err)

	err = api.Fetch(it.ctx, it.config, 200, it.apiOpts...)
	expectErr(it.t, err, state.ErrTargetNotFound)

	err = api.Update(it.ctx, it.config, 200, it.apiOpts...)
	expectErr(it.t, err, state.ErrTargetNotFound)

	// Can't install, no ongoing update
	err = api.Install(it.ctx, it.config, it.apiOpts...)
	expectErr(it.t, err, state.ErrNoUpdateInProgress)

	// Can't start, no ongoing update
	err = api.Start(it.ctx, it.config, it.apiOpts...)
	expectErr(it.t, err, state.ErrNoUpdateInProgress)

	// Non-existing app URI
	target1.Apps[0].PublishedUri = "registry:5000/factory/app-1@sha256:1d36aac9cc6dd3cddeebcf7ad9e98447b73f99f51e273feb29addb0996effbe8"
	it.saveTargetsJson(targets)
	_, _, _ = api.Check(it.ctx, it.config, it.apiOpts...)
	err = api.Fetch(it.ctx, it.config, 100, it.apiOpts...)
	expectErr(it.t, err, state.ErrInitFailed)

	err = api.Update(it.ctx, it.config, -1, it.apiOpts...)
	expectErr(it.t, err, state.ErrInitFailed)
}
