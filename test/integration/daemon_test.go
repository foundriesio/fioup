package integration_tests

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/foundriesio/fioup/pkg/api"
	"github.com/foundriesio/fioup/pkg/state"
	"github.com/stretchr/testify/assert"
)

const (
	maxAttempts int = 3
)

// Verify logic that is specific to the fioup daemon mode
//   - Skipping an update when the target is already running
//   - Retrying a bad target for `maxAttempts` and syncing back to the
//     originally running target after that
func TestUpdateSequenceDaemon(t *testing.T) {
	it := newIntegrationTest(t)

	target1 := it.genNewTarget(100, 2, 50, false, "-stable")
	target2 := it.genNewTarget(101, 3, 60, false, "")
	target3 := it.genNewTarget(102, 1, 70, true, "")

	targets := []*Target{target1}
	it.saveTargetsJson(targets)
	// Update to target1
	err := it.daemonIteration()
	assert.NoError(t, err)
	it.checkStatus(target1.ID, target1.appsURIs(), true)

	// No update should be performed
	err = it.daemonIteration()
	assert.ErrorIs(t, err, state.ErrCheckNoUpdate)

	targets = []*Target{target2}
	it.saveTargetsJson(targets)
	// Update to target2
	err = it.daemonIteration()
	assert.NoError(t, err)
	it.checkStatus(target2.ID, target2.appsURIs(), true)

	// No update should be performed
	err = it.daemonIteration()
	assert.ErrorIs(t, err, state.ErrCheckNoUpdate)

	// No update should be performed
	err = it.daemonIteration()
	assert.ErrorIs(t, err, state.ErrCheckNoUpdate)

	targets = []*Target{target1, target2}
	it.saveTargetsJson(targets)

	// No update should be performed
	err = it.daemonIteration()
	assert.ErrorIs(t, err, state.ErrCheckNoUpdate)

	targets = []*Target{target1, target2, target3}
	it.saveTargetsJson(targets)

	// try installation of target3, which is bad, for maxAttempts times
	for range make([]struct{}, maxAttempts) {
		err = it.daemonIteration()
		assert.ErrorIs(t, err, state.ErrStartFailed)
		it.checkStatus(target2.ID, []string{}, true)
	}

	// Now "rollback" to target2 (do a sync update to target2, re-enabling apps)
	err = it.daemonIteration()
	assert.NoError(t, err)
	it.checkStatus(target2.ID, target2.appsURIs(), true)

	// No update should be performed
	err = it.daemonIteration()
	assert.ErrorIs(t, err, state.ErrCheckNoUpdate)
}

func (it *integrationTest) daemonIteration() error {
	err := api.Update(it.ctx, it.config, -1,
		api.WithGatewayClient(it.gwClient),
		api.WithRequireLatest(true),
		api.WithMaxAttempts(maxAttempts),
		api.WithTUF(false),
	)
	if err != nil && errors.Is(err, state.ErrNewerVersionIsAvailable) {
		slog.Info("Cancelling current update, going to start a new one for the newer version")
		_, err := api.Cancel(it.ctx, it.config)
		if err != nil {
			slog.Error("Error canceling old update", "error", err)
		}
	} else if err != nil && errors.Is(err, state.ErrStartFailed) {
		slog.Info("Error starting updated target", "error", err)
	} else if err != nil && !errors.Is(err, state.ErrCheckNoUpdate) {
		slog.Error("Error during update", "error", err)
	}
	return err
}
