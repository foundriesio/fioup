package integration_tests

import (
	"testing"
)

// Verify that changes in apps list are correctly handled by the update logic
func TestAppsList(t *testing.T) {
	it := newIntegrationTest(t)

	target1 := it.genNewTarget(100, 2, 50, false, "")

	targets := []*Target{target1}
	it.saveTargetsJson(targets)
	it.testUpdateTo(target1, targets)

	it.setApps([]string{"app-1"})
	it.refreshConfig()
	it.testUpdateTo(target1, targets)

	it.setApps([]string{"app-2"})
	it.refreshConfig()
	it.testUpdateTo(target1, targets)

	it.setApps(nil)
	it.refreshConfig()
	it.testUpdateTo(target1, targets)

	it.setApps([]string{})
	it.refreshConfig()
	it.testUpdateTo(target1, targets)

	it.setApps([]string{"app-1", "app-2"})
	it.refreshConfig()
	it.testUpdateTo(target1, targets)
}
