package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents the application configuration.
type Config struct {
	Cookie string `json:"cookie"`
}

// GetConfigPath returns the default configuration file path (~/.config/goyt/config.json).
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "goyt", "config.json"), nil
}

// Load loads the configuration from disk, creating the directory and default file if missing.
func Load() (*Config, error) {
	cfgPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		// Create default empty config
		cfg := &Config{}
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(cfgPath, data, 0644); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
