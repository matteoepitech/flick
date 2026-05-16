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
	"github.com/matteoepitech/flick/internal/cli/config"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// struct used for the JSON template
type Metadata struct {
	Expiration string `json:"expiration"`
}

// createMetadataFile: Creates the metadata file containing the expiration date.
//
// Params:
// - duation (time.Time): The duration of the expiration.
// - filepath (string): The filepath to the metadata location.
// - code (string): The generated share code.
// - logger (logging.Logger): The logger.
func createMetadataFile(duration time.Time, filepath string, code string, logger logging.Logger) {
	meta := Metadata{Expiration: duration.Format(time.RFC3339)}
	data, err := json.Marshal(meta)
	if err != nil {
		logger.InfoError("Failed to create metadata")
	}
	logger.InfoSuccess("Successfully created %s", filepath+"."+code+"-metadata.json")
	os.WriteFile(filepath+"."+code+"-metadata.json", data, 0644)
}

// checkConfigDuration: Checks that the duration set by the user is in config bounds.
//
// Params:
// - duration (time.Time): The duration passed by the user.
//
// Returns:
// - result1 (bool): True if in config bounds, else false.
func checkConfigDuration(duration time.Duration) bool {
	maxExp, _ := time.ParseDuration(config.Conf.MaxExpTime)
	return duration < maxExp
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
	unit := config.Conf.MaxExpTime[len(config.Conf.MaxExpTime)-1]
	value, err := strconv.Atoi(config.Conf.MaxExpTime[:len(config.Conf.MaxExpTime)-1])

	if err != nil {
		logger.InfoError("Non numerical character")
	}
	var maxExp time.Time //TODO: refactor this code to avoid code dupicata
	switch unit {
	case 'd':
		maxExp = time.Now().AddDate(0, 0, value)
	case 'w':
		maxExp = time.Now().AddDate(0, 0, value*7)
	case 'M':
		maxExp = time.Now().AddDate(0, value, 0)
	case 'y':
		maxExp = time.Now().AddDate(value, 0, 0)
	}
	if duration.Before(maxExp) {
		return true
	}
	logger.InfoError("Unsupported time format")
	return false
}

// SetExpiration: Defines the expiration date based on the received pattern.
//
// Params:
// - exp (string): The duration of the expiration.
// - filepath (string): the filepath to the metadata location.
// - code (string): The generated share code.
// - logger (logging.Logger): The logger.
//
// Returns:
// - result1 (bool): Returns true if the metadata file has benn created, else false.
func SetExpiration(exp string, filepath string, code string, logger logging.Logger) bool {
	duration, err := time.ParseDuration(exp)
	if err != nil {
		unit := exp[len(exp)-1]
		value, err := strconv.Atoi(exp[:len(exp)-1])
		if err != nil {
			logger.InfoError("Non numerical character")
		}
		var duration time.Time //TODO: refactor this code to avoid code dupicata
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
		if !checkConfigTime(duration, logger) {
			logger.InfoError("Expiration time higher than maximum defined in configuration")
			return false
		}
		if !duration.IsZero() {
			createMetadataFile(duration, filepath, code, logger)
			return true
		}
		logger.InfoError("Unsupported time format")
		return false
	}
	if !checkConfigDuration(duration) {
		logger.InfoError("Expiration time higher than maximum defined in configuration")
		return false
	}
	createMetadataFile(time.Now().Add(duration), filepath, code, logger)
	return true
}

// CheckExpiration: goroutine that will check and remove every expired files/folders.
//
// Params:
// - dataDir (string): Filapath to the stored files.
// - logger (logging.Logger): The logger.
func CheckExpiration(dataDir string, logger logging.Logger) {
	defTime, _ := time.ParseDuration(config.Conf.DefExpTime) //TODO: check for higher values
	ticker := time.NewTicker(defTime)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			content, err := os.ReadDir(dataDir)
			if err != nil {
				logger.InfoError("Folder %s not found", dataDir)
				return
			}

			logger.Info("Goroutine")
			for _, entries := range content {
				if entries.IsDir() {
					code := entries.Name()
					logger.Info("Opening %s", dataDir+code+"/."+code+"-metadata.json")
					file, err := os.Open(dataDir + code + "/." + code + "-metadata.json")
					if err != nil {
						logger.InfoError("Could not open %s", dataDir+code+"/."+code+"-metadata.json")
						continue
					}
					defer file.Close()

					var meta Metadata
					err = json.NewDecoder(file).Decode(&meta)

					if err != nil {
						logger.InfoError("Could not find expiration time for: %s", code)
					}

					dateExp, err := (time.Parse(time.RFC3339, meta.Expiration))
					if err != nil {
						logger.InfoError("Could not parse expiration time for: %s", code)
					}
					if time.Now().After(dateExp) {
						subdir, _ := os.ReadDir(dataDir + entries.Name())
						for _, files := range subdir {
							logger.Info("Deleting %s", dataDir+code+"/"+files.Name())
							os.Remove(dataDir + code + "/" + files.Name())
						}
						os.Remove(dataDir + entries.Name())
					}
				}
			}
		case <-stop:
			logger.Info("Expiration check stopped")
			return
		}
	}
}
