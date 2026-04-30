/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/download
** File description:
** Download route handler
 */

package routes

import (
	"github.com/matteoepitech/flick/internal/api/logging"
	"io"
	"net/http"
	"os"
)

// DownloadFileHandler: Build the download file handler.
//
// Params:
// - dataDir (string): The directory where files are stored.
// - logger (logging.Logger): The logger to use.
//
// Returns:
// - http.HandlerFunc: The handler function.
func DownloadFileHandler(dataDir string, logger logging.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "This endpoint is meant to be GET only", http.StatusNotFound)
			return
		}

		query := r.URL.Query()
		code := query.Get("code")
		logger.Info("Trying to find the ressource with the code: <%s>", code)
		content, err := os.ReadDir(dataDir + code)

		if err != nil {
			logger.InfoError("Wrong share code")
			return
		}
		for _, entry := range content {
			file, err := os.Stat(dataDir + code + "/" + entry.Name())
			if err != nil {
				logger.InfoError("The ressource <%s> is not found", code)
				http.Error(w, "Resource not found", http.StatusNotFound)
				return
			}
			fileContent, err := os.Open(dataDir + code + "/" + entry.Name())
			if err != nil {
				logger.InfoError("The ressource <%s> can't be downloaded", code)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			io.Copy(w, fileContent)
			logger.InfoSuccess("A file has been downloaded with the code <%s> (%d bytes)", code, file.Size())
		}
	}
}
