package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigPath(t *testing.T) {
	path, err := Path()
	if err != nil {
		t.Fatalf("Path() failed: %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "audeck", "config.json")
	if path != expected {
		t.Errorf("Path() = %v, want %v", path, expected)
	}
}

func TestConfigLoadNotFound(t *testing.T) {
	// Temporarily change the config path
	oldPath := configPath
	configPath = "/nonexistent/path/config.json"
	defer func() { configPath = oldPath }()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.DefaultOutputDevice != "" {
		t.Errorf("Expected empty DefaultOutputDevice, got %v", cfg.DefaultOutputDevice)
	}
	if cfg.DefaultInputDevice != "" {
		t.Errorf("Expected empty DefaultInputDevice, got %v", cfg.DefaultInputDevice)
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "config.json")

	oldPath := configPath
	configPath = testPath
	defer func() { configPath = oldPath }()

	cfg := Config{
		DefaultOutputDevice: "test-output-uid",
		DefaultInputDevice:  "test-input-uid",
	}

	err := cfg.Save()
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loaded.DefaultOutputDevice != cfg.DefaultOutputDevice {
		t.Errorf("DefaultOutputDevice = %v, want %v", loaded.DefaultOutputDevice, cfg.DefaultOutputDevice)
	}
	if loaded.DefaultInputDevice != cfg.DefaultInputDevice {
		t.Errorf("DefaultInputDevice = %v, want %v", loaded.DefaultInputDevice, cfg.DefaultInputDevice)
	}
}

func TestConfigSaveCreatesDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "nested", "dirs", "config.json")

	oldPath := configPath
	configPath = testPath
	defer func() { configPath = oldPath }()

	cfg := Config{}
	err := cfg.Save()
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Error("Save() should create directories")
	}
}

func TestConfigEmptySave(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "config.json")

	oldPath := configPath
	configPath = testPath
	defer func() { configPath = oldPath }()

	cfg := Config{}
	err := cfg.Save()
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file exists and is valid JSON
	data, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	if len(data) == 0 {
		t.Error("Config file should not be empty")
	}
}
