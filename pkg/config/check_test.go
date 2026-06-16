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

	loadConfig := func(watermark, reserved string) *Config {
		setOrDelete := func(key, value string) {
			if len(value) > 0 {
				tree.Set(key, value)
			} else {
				_ = tree.Delete(key)
			}
		}
		setOrDelete(StorageUsageWatermark, watermark)
		setOrDelete(ReservedStorageKey, reserved)
		if b, err := toml.Marshal(tree); err == nil {
			if err := os.WriteFile(tomlConfigPath+"/sota.toml", b, 0644); err != nil {
				t.Fatalf("failed to write temp config file: %v", err)
			}
		} else {
			t.Fatalf("failed to marshal toml tree: %v", err)
		}
		cfg, err := NewConfig([]string{tomlConfigPath})
		checkErr(t, err)
		return cfg
	}
	checkPercent := func(watermark string, expected uint64) {
		cfg := loadConfig(watermark, "")
		if reserved := cfg.GetReservedStorage(); reserved != 0 {
			t.Fatalf("expected no reserved storage for watermark %q, got %d", watermark, reserved)
		}
		if value := cfg.GetStorageWatermark(); value != expected {
			t.Fatalf("expected watermark %d, got %d", expected, value)
		}
	}
	checkReserved := func(watermark, reserved string, expected uint64) {
		cfg := loadConfig(watermark, reserved)
		value := cfg.GetReservedStorage()
		if value == 0 {
			t.Fatalf("expected reserved storage to be set for reserved %q", reserved)
		}
		if value != expected {
			t.Fatalf("expected reserved bytes %d, got %d", expected, value)
		}
	}
	// No value set, should get default
	checkPercent("", StorageUsageWatermarkDefault)
	// Valid percentage
	checkPercent("90", 90)
	// Percentages out of the allowed range fall back to the default
	checkPercent(strconv.Itoa(MinStorageUsageWatermark-1), StorageUsageWatermarkDefault)
	checkPercent(strconv.Itoa(MaxStorageUsageWatermark+1), StorageUsageWatermarkDefault)
	// Invalid percentage falls back to the default
	checkPercent("80abc", StorageUsageWatermarkDefault)
	// reserved_storage sets an absolute reserved free space in bytes
	checkReserved("", "2GiB", 2*1024*1024*1024)
	checkReserved("", "500MiB", 500*1024*1024)
	// decimal byte suffixes are accepted in addition to the binary ones
	checkReserved("", "2GB", 2*1000*1000*1000)
	checkReserved("", "500MB", 500*1000*1000)
	// reserved_storage takes precedence over storage_watermark when both are set
	checkReserved("80", "2GiB", 2*1024*1024*1024)
	// Invalid reserved_storage is ignored, falling back to the percentage watermark
	cfg := loadConfig("90", "2XB")
	if reserved := cfg.GetReservedStorage(); reserved != 0 {
		t.Fatalf("expected no reserved storage when reserved_storage is invalid, got %d", reserved)
	}
	if value := cfg.GetStorageWatermark(); value != 90 {
		t.Fatalf("expected percentage watermark 90 when reserved_storage is invalid, got %d", value)
	}
}
