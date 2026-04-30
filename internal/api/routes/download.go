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
	"mime/multipart"
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
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		path := dataDir + code

		entries, err := os.ReadDir(path)
		if err != nil {
			logger.InfoError("The code <%s> doesn't exist.", code)
			http.Error(w, "Code not found", http.StatusNotFound)
			return
		}

		writer := multipart.NewWriter(w)
		defer writer.Close()

		w.Header().Set("Content-Type", writer.FormDataContentType())

		for _, entry := range entries {
			fullPath := path + "/" + entry.Name()

			file, err := os.Open(fullPath)
			if err != nil {
				logger.InfoError("Cannot open file %s", entry.Name())
				continue
			}

			info, err := file.Stat()
			if err != nil {
				file.Close()
				continue
			}

			part, err := writer.CreateFormFile("file", entry.Name())
			if err != nil {
				file.Close()
				continue
			}

			_, err = io.Copy(part, file)
			file.Close()

			if err != nil {
				continue
			}

			logger.InfoSuccess("Sent file <%s> (%d bytes)", entry.Name(), info.Size())
		}
	}
}
