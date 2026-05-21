/*
** FLICK PROJECT, 2026
** flick/internal/cli/config/configuration
** File description:
** Configuration source file
 */

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Configuration structure type
type Configuration struct {
	ServerIP         string `json:"server_ip"`
	DefExpTime       string `json:"def_exp_time"`
	DefDownloadCount int32  `json:"def_download_count"`
}

// Global configuration of the CLI
var Conf Configuration = Configuration{
	ServerIP:         "127.0.0.1",
	DefExpTime:       "15m",
	DefDownloadCount: 1,
}

// configPaths: Resolve the configuration directory and file paths.
//
// Returns:
// - result1 (string): The configuration directory.
// - result2 (string): The configuration file path.
// - result3 (error): If something occured.
func configPaths() (string, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve home directory: %w", err)
	}
	dir := filepath.Join(homeDir, ".flick")
	return dir, filepath.Join(dir, "config.json"), nil
}

// LoadWithFile: Load the configuration file at ~/.flick/config.json.
//
// Returns:
// - result1 (error): If something occured.
func (c *Configuration) LoadWithFile() error {
	_, configFile, err := configPaths()
	if err != nil {
		return err
	}

	file, err := os.Open(configFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return createConfigurationFile()
		}
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(c); err != nil {
		return fmt.Errorf("failed to decode config file: %w", err)
	}
	return nil
}

// SaveConfigurationFile: Save the configuration file at ~/.flick/config.json.
//
// Returns:
// - result1 (error): If something occured.
func (c Configuration) SaveConfigurationFile() error {
	configDir, configFile, err := configPaths()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.Create(configFile)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")

	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed to encode config file: %w", err)
	}
	return nil
}

// createConfigurationFile: Create the configuration file at ~/.flick/config.json with defaults.
//
// Returns:
// - result1 (error): If something occured.
func createConfigurationFile() error {
	return Conf.SaveConfigurationFile()
}
