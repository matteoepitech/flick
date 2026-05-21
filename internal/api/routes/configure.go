/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/configure
** File description:
** Configuration route
 */

package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/matteoepitech/flick/internal/api/logging"
	"github.com/matteoepitech/flick/internal/api/path"
	"github.com/matteoepitech/flick/internal/api/serverconfig"
	"github.com/matteoepitech/flick/internal/api/utils"
)

// WriteDefaultConfig: Writes the default server configuration.
//
// Params:
// - logger (logging.Logger): The logger.
func WriteDefaultConfig(logger logging.Logger) {
	dir := path.GetFlickDir()
	if _, err := os.Stat(filepath.Join(dir, "server-config.json")); err == nil {
		logger.Info("Server configuration file already exists")
		return
	}
	data, _ := json.MarshalIndent(serverconfig.Conf, "", "")
	os.WriteFile(filepath.Join(dir, "server-config.json"), data, 0644)
}

// SendServerConfig: Sends the server config to the web if GET,
// if POST modifies the server config.
//
// Params:
// - logger (logging.Logger): The logger.
//
// Returns:
// - http.HandlerFunc
func SendServerConfig(logger logging.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dir := path.GetFlickDir()

		if r.Method != http.MethodPost {
			data, _ := os.ReadFile(filepath.Join(dir, "server-config.json"))
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
			return
		}

		var newConf serverconfig.Configuration
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&newConf); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		if err := serverconfig.Validate(&newConf); err != nil {
			http.Error(w, "Validation failed: "+err.Error(), http.StatusBadRequest)
			return
		}

		defaultDur, _ := utils.ParseExpirationTime(newConf.DefaultExpiration)
		maxDur, _ := utils.ParseExpirationTime(newConf.MaxExpiration)
		if defaultDur.After(maxDur) {
			http.Error(w, "default_expiration must be <= max_expiration", http.StatusBadRequest)
			return
		}

		serverconfig.Conf = newConf
		data, _ := json.MarshalIndent(serverconfig.Conf, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "server-config.json"), data, 0644); err != nil {
			http.Error(w, "Failed to save config", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, "OK")
	}
}
