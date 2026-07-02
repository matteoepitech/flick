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

	"github.com/Flick-Corp/flick/internal/api/path"
	"github.com/Flick-Corp/flick/internal/api/serverconfig"
)

// ServerUserConfigHandler: Send the user config.
// The server-config.json contains field that the user can read.
// We filter the fields and send the results.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func ServerUserConfigHandler() http.HandlerFunc {
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

			out, _ := json.MarshalIndent(serverconfig.FilterUserFields(conf), "", "  ")
			w.Header().Set("Content-Type", "application/json")
			w.Write(out)
			return
		}

		data, err := os.ReadFile(filepath.Join(path.GetFlickDir(), "server-config.json"))
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "Failed to read config")
			return
		}

		var conf serverconfig.Configuration
		if err := json.Unmarshal(data, &conf); err != nil {
			WriteError(w, http.StatusInternalServerError, "Failed to parse config")
			return
		}

		out, err := json.MarshalIndent(serverconfig.UserFields(conf), "", "  ")
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "Failed to encode config")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	}
}
