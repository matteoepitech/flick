/*
** FLICK PROJECT, 2026
** flick/internal/cli/config/configuration
** File description:
** Configuration source file
 */

package config

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// Configuration structure type
type Configuration struct {
	ServerIP         string `json:"server_ip"`
	DefExpTime       string `json:"default_expiration"`
	DefDownloadCount int32  `json:"default_download_count"`
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
		return fmt.Errorf("Failed to create config directory: %w", err)
	}

	file, err := os.Create(configFile)
	if err != nil {
		return fmt.Errorf("Failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")

	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("Failed to encode config file: %w", err)
	}
	return nil
}

// ReplaceUsingServerConfiguration: Replace the content of the default configuration of server against the current client configuration.
//
// Returns:
// - result1 (error): If something occured.
func (c *Configuration) ReplaceUsingServerConfiguration() error {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Dev only: local self-signed cert.
		},
	}
	url := fmt.Sprintf("https://%s:15702/user-configure", c.ServerIP)

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("Failed to fetch server configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Server returned %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(c); err != nil {
		return fmt.Errorf("Failed to decode server configuration: %w", err)
	}
	return nil
}

// createConfigurationFile: Create the configuration file at ~/.flick/config.json with defaults.
//
// Returns:
// - result1 (error): If something occured.
func createConfigurationFile() error {
	if err := Conf.ReplaceUsingServerConfiguration(); err != nil {
		return fmt.Errorf("Failed to retrieve configuration from server")
	}

	if err := Conf.SaveConfigurationFile(); err != nil {
		return fmt.Errorf("Failed to create configuration file at ~/.flick/config.json")
	}
	return nil
}
