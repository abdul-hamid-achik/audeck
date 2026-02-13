package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds user-persisted preferences.
type Config struct {
	// DefaultOutputDevice is the preferred default output device UID.
	DefaultOutputDevice string `json:"default_output_device,omitempty"`
	// DefaultInputDevice is the preferred default input device UID.
	DefaultInputDevice string `json:"default_input_device,omitempty"`
}

// Path returns the default config file path (~/.config/audeck/config.json).
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "audeck", "config.json"), nil
}

// Load reads the config from disk. Returns a zero Config if the file
// does not exist.
func Load() (Config, error) {
	p, err := Path()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	return c, json.Unmarshal(data, &c)
}

// Save writes the config to disk, creating directories as needed.
func (c Config) Save() error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}
