package config

import (
	"os"
	"strconv"
	"testing"

	"github.com/pelletier/go-toml"
)

func checkErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfig_StorageWatermark(t *testing.T) {
	tomlConfigPath := t.TempDir()
	tree, err := toml.TreeFromMap(nil)
	checkErr(t, err)
	tree.Set(ServerBaseUrlKey, "https://updates.example.com")
	tree.Set("pacman.reset_apps_root", tomlConfigPath)
	tree.Set("pacman.compose_apps_root", tomlConfigPath)
	tree.Set("storage.path", tomlConfigPath)

	checkStorageWatermark := func(value string, expected uint) {
		if len(value) > 0 {
			tree.Set(StorageUsageWatermark, value)
		}
		if b, err := toml.Marshal(tree); err == nil {
			if err := os.WriteFile(tomlConfigPath+"/sota.toml", b, 0644); err != nil {
				t.Fatalf("failed to write temp config file: %v", err)
			}
		} else {
			t.Fatalf("failed to marshal toml tree: %v", err)
		}
		cfg, err := NewConfig([]string{tomlConfigPath})
		checkErr(t, err)
		if cfg.GetStorageUsageWatermark() != expected {
			t.Fatalf("expected watermark %d, got %d", expected, cfg.GetStorageUsageWatermark())
		}
		if _, ok := cfg.GetStorageReservedBytes(); ok {
			t.Fatalf("expected percentage mode for value %q, got reserved-bytes mode", value)
		}
	}
	checkReservedBytes := func(value string, expectedBytes uint64) {
		tree.Set(StorageUsageWatermark, value)
		if b, err := toml.Marshal(tree); err == nil {
			if err := os.WriteFile(tomlConfigPath+"/sota.toml", b, 0644); err != nil {
				t.Fatalf("failed to write temp config file: %v", err)
			}
		} else {
			t.Fatalf("failed to marshal toml tree: %v", err)
		}
		cfg, err := NewConfig([]string{tomlConfigPath})
		checkErr(t, err)
		reserved, ok := cfg.GetStorageReservedBytes()
		if !ok {
			t.Fatalf("expected reserved-bytes mode for value %q", value)
		}
		if reserved != expectedBytes {
			t.Fatalf("expected reserved bytes %d, got %d", expectedBytes, reserved)
		}
	}
	// No value set, should get default
	checkStorageWatermark("", StorageUsageWatermarkDefault)
	// Valid value
	checkStorageWatermark("90", 90)
	// Values out of the allowed range
	checkStorageWatermark(strconv.Itoa(MinStorageUsageWatermark-1), StorageUsageWatermarkDefault)
	checkStorageWatermark(strconv.Itoa(MaxStorageUsageWatermark+1), StorageUsageWatermarkDefault)
	// Invalid value
	checkStorageWatermark("80abc", StorageUsageWatermarkDefault)
	// Invalid size suffix falls back to the default percentage
	checkStorageWatermark("2XB", StorageUsageWatermarkDefault)
	// Absolute reserved free space via size suffix
	checkReservedBytes("2GiB", 2*1024*1024*1024)
	checkReservedBytes("500MiB", 500*1024*1024)
}
