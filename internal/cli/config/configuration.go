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
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/matteoepitech/flick/internal/cli/network"
)

// Configuration structure type
type Configuration struct {
	ServerURL        string `json:"server_url"`
	DefExpTime       string `json:"default_expiration"`
	DefDownloadCount int32  `json:"default_download_count"`
}

// Server limits structure type (fetched dynamically, not saved in local config)
type ServerLimits struct {
	MaxFileSizeMb    int32  `json:"max_file_size_mb"`
	MaxExpiration    string `json:"max_expiration"`
	MaxDownloadCount int32  `json:"max_download_count"`
}

// Global configuration of the CLI
var Conf Configuration = Configuration{
	ServerURL:        "http://localhost",
	DefExpTime:       "15m",
	DefDownloadCount: 1,
}

// NormalizeServerURL: Normalize a server URL: default to https:// when no
// scheme is given and strip any trailing slash.
//
// Params:
// - raw (string): The raw URL as typed by the user.
//
// Returns:
// - result1 (string): The normalized URL.
func NormalizeServerURL(raw string) string {
	serverURL := strings.TrimRight(strings.TrimSpace(raw), "/")
	if serverURL != "" && !strings.Contains(serverURL, "://") {
		serverURL = "https://" + serverURL
	}
	return serverURL
}

// APIBaseURL: Build the base URL of the API routes (server URL + /api/v1).
//
// Returns:
// - result1 (string): The base URL, without trailing slash.
func (c Configuration) APIBaseURL() string {
	return NormalizeServerURL(c.ServerURL) + "/api/v1"
}

// GetServerLimits: Get the current server limits and configuration.
//
// Returns:
// - result1 (*ServerLimits): The server limits.
// - result2 (error): If something occured.
func GetServerLimits() (*ServerLimits, error) {
	resp, err := network.SharedClient.Get(Conf.APIBaseURL() + "/user-configure")
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch server configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Server returned %s", resp.Status)
	}

	var limits ServerLimits
	if err := json.NewDecoder(resp.Body).Decode(&limits); err != nil {
		return nil, fmt.Errorf("Failed to decode server configuration: %w", err)
	}
	return &limits, nil
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
	resp, err := network.SharedClient.Get(c.APIBaseURL() + "/user-configure")
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
