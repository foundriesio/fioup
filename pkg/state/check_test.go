package state

import (
	"strings"
	"testing"

	"github.com/foundriesio/composeapp/pkg/update"
)

func TestCheck_VersionExtraction(t *testing.T) {
	{
		//  Positive test case
		v, err := extractTargetVersion("arm64-linux-21")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if v != 21 {
			t.Fatalf("Expected version 21, got %d", v)
		}
	}
	{
		// Negative test case
		v, err := extractTargetVersion("arm64-linux-21-main")
		if err == nil {
			t.Fatalf("Expected error, got version %d", v)
		}
		if v != -1 {
			t.Fatalf("Expected version -1 on error, got %d", v)
		}
	}
}

func TestCheck_GetTargetOutOfUpdate(t *testing.T) {
	appURIs := []string{
		"registry.io/repo/app@sha256:c8087fe1b69ccc025c6cedb8747d98e132d7bf8be8081cb2403ebd3b6545ed6a",
	}
	{
		// Positive test case
		u := &update.Update{
			ClientRef: "arm64-linux-42",
			URIs:      appURIs,
		}
		target, err := getTargetOutOfUpdate(u)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if target == nil {
			t.Fatalf("Expected non-nil target")
		}
		if target.Version != 42 {
			t.Fatalf("Expected version 42, got %d", target.Version)
		}
		if len(target.Apps) != 1 {
			t.Fatalf("Expected 1 app, got %d", len(target.Apps))
		}
		if target.Apps[0].URI != appURIs[0] {
			t.Fatalf("Expected app URI %s, got %s", appURIs[0], target.Apps[0].URI)
		}
	}
	{
		// Negative test case
		u := &update.Update{
			ClientRef: "arm64-linux-42-main",
			URIs:      appURIs,
		}
		target, err := getTargetOutOfUpdate(u)
		if err == nil {
			t.Fatalf("Expected error, got target: %+v", target)
		}
		if !strings.Contains(err.Error(), "failed to extract target version") {
			t.Fatalf("Unexpected error message: %v", err)
		}
		if target != nil {
			t.Fatalf("Expected nil target on error, got: %+v", target)
		}
	}
}
