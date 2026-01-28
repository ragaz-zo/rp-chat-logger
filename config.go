package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// saveConfiguration writes the application config to a JSON file
// in the user's config directory.
func saveConfiguration(config *AppConfig) error {
	configPath := getConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	file, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("opening config file for writing: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	return nil
}

// loadConfiguration reads the application config from a JSON file
// in the user's config directory.
func loadConfiguration() (*AppConfig, error) {
	file, err := os.Open(getConfigPath())
	if err != nil {
		return nil, fmt.Errorf("opening config file: %w", err)
	}
	defer file.Close()

	config := &AppConfig{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("decoding config: %w", err)
	}
	return config, nil
}

// getConfigPath returns the full path to the application config file.
func getConfigPath() string {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join("rp-chat-logger", "config.json")
	}
	return filepath.Join(userConfigDir, "rp-chat-logger", "config.json")
}
