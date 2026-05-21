/*
** FLICK PROJECT, 2026
** flick/internal/api/metadata/metadata.go
** File description:
** metadata.go
 */

package metadata

import (
	"encoding/json"
	"os"
	"time"

	"github.com/matteoepitech/flick/internal/api/logging"
	"github.com/matteoepitech/flick/internal/api/serverconfig"
	"github.com/matteoepitech/flick/internal/api/utils"
)

// struct used for the JSON template
type Metadata struct {
	Expiration string `json:"expiration"`
}

// createMetadataFile: Creates the metadata file containing the expiration date.
//
// Params:
// - metadata (Metadata): The metadata informations.
// - filepath (string): The filepath to the metadata location.
// - code (string): The generated share code.
// - logger (logging.Logger): The logger.
func CreateMetadataFile(metadata Metadata, filepath string, code string, logger logging.Logger) {
	data, err := json.Marshal(metadata)
	if err != nil {
		logger.InfoError("Failed to create metadata")
	}

	logger.InfoSuccess("Successfully created %s", filepath+"."+code+"-metadata.json")
	os.WriteFile(filepath+"."+code+"-metadata.json", data, 0644)
}

// SetExpiration: Defines the expiration date based on the received pattern.
//
// Params:
// - metadata (*Metadata): The metadata to set the expiration.
// - exp (string): The duration of the expiration.
// - logger (logging.Logger): The logger.
//
// Returns:
// - result1 (bool): Returns true if the metadata file has benn created, else false.
func SetExpiration(metadata *Metadata, exp string, logger logging.Logger) bool {
	duration, err := utils.ParseExpirationTime(exp)

	if err != nil {
		logger.InfoError("Failed to parse expiration time")
		return false
	}

	if !duration.After(time.Now()) {
		logger.InfoError("Expiration time is before now, cannot set the expiration time")
		return false
	}
	if !checkConfigTime(duration, logger) {
		logger.InfoError("Expiration time higher than maximum defined in configuration (%q)", duration)
		return false
	}
	if duration.IsZero() {
		logger.InfoError("Expiration time is zero, cannot set the expiration time")
		return false
	}
	metadata.Expiration = duration.Format(time.RFC3339)
	return true
}

// CheckExpirationToRemove: Will check and remove every expired files/folders.
//
// Params:
// - dataDir (string): The data directory.
//
// Returns:
// - result1 (error): An error if occured.
func CheckExpirationToRemove(dataDir string) error {
	content, err := os.ReadDir(dataDir)
	if err != nil {
		return err
	}

	for _, entries := range content {
		if entries.IsDir() {
			code := entries.Name()
			file, err := os.Open(dataDir + code + "/." + code + "-metadata.json")
			if err != nil {
				continue
			}
			defer file.Close()

			var meta Metadata
			err = json.NewDecoder(file).Decode(&meta)

			if err != nil {
				continue
			}

			dateExp, err := (time.Parse(time.RFC3339, meta.Expiration))
			if err != nil {
				continue
			}
			if time.Now().After(dateExp) {
				subdir, _ := os.ReadDir(dataDir + entries.Name())
				for _, files := range subdir {
					os.Remove(dataDir + code + "/" + files.Name())
				}
				os.Remove(dataDir + entries.Name())
			}
		}
	}
	return nil
}

// checkConfigTime: Checks that the duration set by the user is in config bounds.
//
// Params:
// - duration (time.Time): The duration passed by the user.
// - logger (logging.Logger): The logger.
//
// Returns:
// - result1 (bool): True if in config bounds, else false.
func checkConfigTime(duration time.Time, logger logging.Logger) bool {
	maxExp, err := utils.ParseExpirationTime(serverconfig.Conf.MaxExpiration)
	if err != nil {
		logger.InfoError("Failed to parse max expiration time in configuration")
		return false
	}

	return duration.Before(maxExp.Add(time.Second))
}
