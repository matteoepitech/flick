/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/user_configure
** File description:
** Configuration route but for user-facing settings
 */

package routes

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/matteoepitech/flick/internal/api/logging"
	"github.com/matteoepitech/flick/internal/api/path"
	"github.com/matteoepitech/flick/internal/api/serverconfig"
)

// SendServerUserConfig: Sends the user-facing server config to the web by a GET.
// Only the fields tagged with `user:"true"` are returned.
//
// Params:
// - logger (logging.Logger): The logger.
//
// Returns:
// - http.HandlerFunc
func SendServerUserConfig(logger logging.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dir := path.GetFlickDir()

		// GET method
		if r.Method == http.MethodGet {
			data, err := os.ReadFile(filepath.Join(dir, "server-config.json"))
			if err != nil {
				http.Error(w, "Failed to read config", http.StatusInternalServerError)
				return
			}

			var conf serverconfig.Configuration
			if err := json.Unmarshal(data, &conf); err != nil {
				http.Error(w, "Failed to parse config", http.StatusInternalServerError)
				return
			}

			out, _ := json.MarshalIndent(serverconfig.UserFields(conf), "", "  ")
			w.Header().Set("Content-Type", "application/json")
			w.Write(out)
			return
		}

		http.Error(w, "Invalid request", http.StatusBadRequest)
	}
}
