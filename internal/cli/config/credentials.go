/*
** FLICK PROJECT, 2026
** flick/internal/cli/config/credentials
** File description:
** Credentials source file
 */

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/matteoepitech/flick/internal/cli/network"
)

// Credentials structure type, stored at ~/.flick/credentials.json
type Credentials struct {
	UserID string `json:"user_id"`
	Token  string `json:"token"`
}

// credentialsPath: Resolve the credentials file path.
//
// Returns:
// - result1 (string): The credentials file path.
// - result2 (error): If something occured.
func credentialsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve home directory: %w", err)
	}
	return filepath.Join(homeDir, ".flick", "credentials.json"), nil
}

// loadCredentials: Load the credentials file at ~/.flick/credentials.json.
//
// Returns:
// - result1 (*Credentials): The credentials, nil if the file does not exist.
// - result2 (error): If something occured.
func LoadCredentials() (*Credentials, error) {
	credentialsFile, err := credentialsPath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(credentialsFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open credentials file: %w", err)
	}
	defer file.Close()

	creds := &Credentials{}
	if err := json.NewDecoder(file).Decode(creds); err != nil {
		return nil, fmt.Errorf("failed to decode credentials file: %w", err)
	}
	return creds, nil
}

// saveCredentials: Save the credentials file at ~/.flick/credentials.json.
//
// Params:
// - creds (Credentials): The credentials to save.
//
// Returns:
// - result1 (error): If something occured.
func SaveCredentials(creds Credentials) error {
	credentialsFile, err := credentialsPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(credentialsFile), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.OpenFile(credentialsFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create credentials file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")

	if err := encoder.Encode(creds); err != nil {
		return fmt.Errorf("failed to encode credentials file: %w", err)
	}
	return nil
}

// identifyOnServer: Ask the server to create an anonymous user and return the
// associated credentials.
//
// Returns:
// - result1 (*Credentials): The new credentials.
// - result2 (error): If something occured.
func identifyOnServer() (*Credentials, error) {
	resp, err := network.SharedClient.Post(Conf.APIBaseURL()+"/identify", "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to identify on the server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("server returned %s", resp.Status)
	}

	creds := &Credentials{}
	if err := json.NewDecoder(resp.Body).Decode(creds); err != nil {
		return nil, fmt.Errorf("failed to decode identify response: %w", err)
	}
	if creds.UserID == "" {
		return nil, fmt.Errorf("server returned an empty user id")
	}
	return creds, nil
}

// EnsureCredentials: Return the local credentials, asking the server to
// create an anonymous user (and saving the result) when none exist yet.
//
// Returns:
// - result1 (*Credentials): The credentials.
// - result2 (error): If something occured.
func EnsureCredentials() (*Credentials, error) {
	creds, err := LoadCredentials()
	if err != nil {
		return nil, err
	}
	if creds != nil && creds.UserID != "" {
		return creds, nil
	}

	creds, err = identifyOnServer()
	if err != nil {
		return nil, err
	}

	if err := SaveCredentials(*creds); err != nil {
		return nil, err
	}
	return creds, nil
}
