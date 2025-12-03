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
}
