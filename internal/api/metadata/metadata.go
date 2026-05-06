/*
** FLICK PROJECT, 2026
** flick/internal/api/metadata/metadata.go
** File description:
** metadata.go
 */

package metadata

import (
	"encoding/json"
	"github.com/matteoepitech/flick/internal/api/logging"
	"os"
	"strconv"
	"time"
)

// struct used for the JSON template
type Metadata struct {
	Expiration string `json:"expiration"`
}

// createMetadataFile: Creates the metadata file.
//
// Params:
// - duation (time.Time): The duration of the expiration.
// - filepath (string): the filepath to the metadata location.
// - code (string): The generated share code.
// - logger (logging.Logger): The logger.
func createMetadataFile(duration time.Time, filepath string, code string, logger logging.Logger) {
	meta := Metadata{Expiration: duration.Format(time.RFC3339)}
	data, err := json.Marshal(meta)
	if err != nil {
		logger.InfoError("Failed to create metadata")
	}
	logger.InfoSuccess("%s", filepath+"."+code+"-metadata.json")
	os.WriteFile(filepath+code+"-metadata.json", data, 0644)
}

// SetExpiration: Defines the expiration date based on the received pattern.
//
// Params:
// - exp (string): The duration of the expiration.
// - filepath (string): the filepath to the metadata location.
// - code (string): The generated share code.
// - logger (logging.Logger): The logger.
func SetExpiration(exp string, filepath string, code string, logger logging.Logger) {
	duration, err := time.ParseDuration(exp)
	if err != nil {
		unit := exp[len(exp)-1]
		value, err := strconv.Atoi(exp[:len(exp)-1])
		if err != nil {
			logger.InfoError("Non numerical character")
		}
		var duration time.Time
		switch unit {
		case 'd':
			duration = time.Now().AddDate(0, 0, value)
		case 'w':
			duration = time.Now().AddDate(0, 0, value*7)
		case 'M':
			duration = time.Now().AddDate(0, value, 0)
		case 'y':
			duration = time.Now().AddDate(value, 0, 0)
		}
		if !duration.IsZero() {
			createMetadataFile(duration, filepath, code, logger)
		}
		logger.InfoError("Unsupported time format")
		return
	}
	createMetadataFile(time.Now().Add(duration), filepath, code, logger)
}

// CheckExpiration: goroutine that will check and remove every expired files/folders.
//
// Params:
// - dataDir (string): Filapath to the stored files.
// - logger (logging.Logger): The logger.
func CheckExpiration(dataDir string, logger logging.Logger) {
	ticker := time.NewTicker(1 * time.Hour)
	stop := make(chan bool)
	defer ticker.Stop()

	content, err := os.ReadDir(dataDir)
	if err != nil {
		logger.InfoError("Folder not found")
		return
	}
	for {
		select {
		case <-ticker.C:
			for _, entries := range content {
				if entries.IsDir() {
					code := entries.Name()

					file, err := os.Open(dataDir + code + "/." + code + "-metadata.json")
					if err != nil {
						return
					}
					defer file.Close()

					var meta Metadata
					err = json.NewDecoder(file).Decode(&meta)
					if err != nil {
						return
					}

					dateExp, err := (time.Parse(time.RFC3339, meta.Expiration))
					if err != nil {
						return
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
		case <-stop:
			os.Exit(0)
			return
		}
	}
}
