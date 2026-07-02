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

	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/path"
	"github.com/Flick-Corp/flick/internal/api/serverconfig"
	"github.com/Flick-Corp/flick/internal/api/utils"
)

// SendServerConfig: Sends the server config to the web if GET,
// if POST modifies the server config.
//
// Returns:
// - http.HandlerFunc
func SendServerConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dir := path.GetFlickDir()

		// GET method
		if r.Method == http.MethodGet {
			data, err := os.ReadFile(filepath.Join(dir, "server-config.json"))
			if err != nil {
				WriteError(w, http.StatusInternalServerError, "Failed to read config")
				return
			}

			var conf serverconfig.Configuration
			if err := json.Unmarshal(data, &conf); err != nil {
				WriteError(w, http.StatusInternalServerError, "Failed to parse config")
				return
			}

			out, _ := json.MarshalIndent(conf, "", "  ")
			w.Header().Set("Content-Type", "application/json")
			w.Write(out)

			return
		}

		// POST method
		if r.Method == http.MethodPost {
			var newConf serverconfig.Configuration
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&newConf); err != nil {
				WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
				return
			}

			if err := newConf.Validate(); err != nil {
				WriteError(w, http.StatusBadRequest, "Validation failed: "+err.Error())
				return
			}

			defaultDur, _ := utils.ParseExpirationTime(newConf.DefaultExpiration)
			maxDur, _ := utils.ParseExpirationTime(newConf.MaxExpiration)
			if defaultDur.After(maxDur) {
				WriteError(w, http.StatusBadRequest, "default_expiration must be <= max_expiration")
				return
			}

			serverconfig.Conf = newConf
			data, _ := json.MarshalIndent(serverconfig.Conf, "", "  ")
			if err := os.WriteFile(filepath.Join(dir, "server-config.json"), data, 0644); err != nil {
				logging.LogInfoError("Cannot save server configuration: %v", err)
				WriteError(w, http.StatusInternalServerError, "Failed to save config")
				return
			}
			logging.LogInfoSuccess("Server configuration updated")
			fmt.Fprint(w, "OK")
			return
		}

		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}
