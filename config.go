package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func saveConfiguration(config *AppConfig) error {
	file, err := os.OpenFile(getConfigPath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(config)
}

func loadConfiguration() (*AppConfig, error) {
	file, err := os.Open(getConfigPath())
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &AppConfig{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return nil, err
	}
	return config, nil
}

func getConfigPath() string {
	configFileName := "app_config.json"
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to current directory if user config dir cannot be determined
		return configFileName
	}
	return filepath.Join(userConfigDir, configFileName)
}
