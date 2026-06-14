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
	"strconv"
	"time"

	"github.com/matteoepitech/flick/internal/api/logging"
	"github.com/matteoepitech/flick/internal/api/serverconfig"
	"github.com/matteoepitech/flick/internal/api/utils"
	"github.com/matteoepitech/flick/internal/api/utils/data"
)

// struct used for the JSON template
type Metadata struct {
	Expiration           string `json:"expiration"`
	CurrentDownloadCount int32  `json:"current_download_count"`
	MaxDownloadCount     int32  `json:"max_download_count"`
	UploaderID           string `json:"uploader_id,omitempty"`
}

// createMetadataFile: Creates the metadata file containing the expiration date.
//
// Params:
// - metadata (Metadata): The metadata informations.
// - filepath (string): The filepath to the metadata location.
// - code (string): The generated share code.
func CreateMetadataFile(metadata Metadata, filepath string, code string) {
	metadataPath := filepath + "." + code + "-metadata.json"

	data, err := json.Marshal(metadata)
	if err != nil {
		logging.LogInfoError("Cannot marshal metadata for code %q: %v", code, err)
		return
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		logging.LogInfoError("Cannot write metadata file %q: %v", metadataPath, err)
		return
	}

	logging.LogInfoSuccess("Created metadata file %q", metadataPath)
}

// SetExpiration: Defines the expiration date based on the received pattern.
//
// Params:
// - metadata (*Metadata): The metadata to set the expiration.
// - exp (string): The duration of the expiration.
//
// Returns:
// - result1 (bool): Return true if the metadata has been changed, else false.
func SetExpiration(metadata *Metadata, exp string) bool {
	duration, err := utils.ParseExpirationTime(exp)

	if err != nil {
		logging.LogInfoError("Cannot parse expiration time %q: %v", exp, err)
		return false
	}

	if duration.IsZero() {
		logging.LogInfoError("Expiration time %q cannot be zero", exp)
		return false
	}
	if !duration.After(time.Now()) {
		logging.LogInfoError("Expiration time %q is in the past", exp)
		return false
	}
	if !checkConfigTime(duration) {
		logging.LogInfoError("Expiration time %q exceeds the maximum allowed by configuration", exp)
		return false
	}
	metadata.Expiration = duration.Format(time.RFC3339)
	return true
}

// SetMaxDownloadCount: Defines the max download count based on the received pattern.
//
// Params:
// - metadata (*Metadata): The metadata to modify.
// - maxDownloadCount (string): The max download count string.
//
// Returns:
// - result1 (bool): Return true if the metadata has been changed, else false.
func SetMaxDownloadCount(metadata *Metadata, maxDownloadCount string) bool {
	mdc, err := strconv.Atoi(maxDownloadCount)
	if err != nil {
		logging.LogInfoError("Cannot parse max download count %q: %v", maxDownloadCount, err)
		return false
	}

	if mdc > serverconfig.Conf.MaxDownloadCount {
		logging.LogInfoError("Max download count %q exceeds the maximum allowed by configuration (%d)", maxDownloadCount, serverconfig.Conf.MaxDownloadCount)
		return false
	}

	metadata.MaxDownloadCount = int32(mdc)
	return true
}

// SetUploaderID: Defines the uploader id. The uploader is mandatory, so an empty
// id is rejected. The id is expected to be already validated against the
// database by the caller.
//
// Params:
// - metadata (*Metadata): The metadata to modify.
// - uploaderID (string): The validated uploader UUID.
//
// Returns:
// - result1 (bool): Return true if the metadata has been changed, else false.
func SetUploaderID(metadata *Metadata, uploaderID string) bool {
	if uploaderID == "" {
		logging.LogInfoError("Uploader id is required")
		return false
	}

	metadata.UploaderID = uploaderID
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
				data.DeleteDataDirWithCode(entries.Name())
			}
		}
	}
	return nil
}

// checkConfigTime: Checks that the duration set by the user is in config bounds.
//
// Params:
// - duration (time.Time): The duration passed by the user.
//
// Returns:
// - result1 (bool): True if in config bounds, else false.
func checkConfigTime(duration time.Time) bool {
	maxExp, err := utils.ParseExpirationTime(serverconfig.Conf.MaxExpiration)
	if err != nil {
		logging.LogInfoError("Cannot parse max expiration time %q from configuration: %v", serverconfig.Conf.MaxExpiration, err)
		return false
	}

	return duration.Before(maxExp.Add(time.Second))
}
