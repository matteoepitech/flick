/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/download
** File description:
** Download route handler
 */

package routes

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	"github.com/matteoepitech/flick/internal/api/logging"
	"github.com/matteoepitech/flick/internal/api/path"
)

// doDownloadMultipartForm: Do the download request.
//
// Params:
// - writer (*multipart.Writer): The writer of the multipart.
// - entries ([]os.DirEntry): The different entries of the directory.
// - path (string): The path.
// - logger (logging.Logger): The logger.
func doDownloadMultipartForm(writer *multipart.Writer, entries []os.DirEntry, path string, logger logging.Logger) {
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

// DownloadFileHandler: Build the download file handler.
//
// Params:
// - logger (logging.Logger): The logger to use.
//
// Returns:
// - http.HandlerFunc: The handler function.
func DownloadFileHandler(logger logging.Logger) http.HandlerFunc {
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

		codePath := path.GetDataDir() + code

		entries, err := os.ReadDir(codePath)
		if err != nil {
			logger.InfoError("The code <%s> doesn't exist.", code)
			http.Error(w, "Code not found", http.StatusNotFound)
			return
		}

		var totalSize int64
		for _, entry := range entries {
			if info, err := entry.Info(); err == nil {
				totalSize += info.Size()
			}
		}

		writer := multipart.NewWriter(w)
		defer writer.Close()

		w.Header().Set("Content-Type", writer.FormDataContentType())
		w.Header().Set("X-Total-Size", strconv.FormatInt(totalSize, 10))
		doDownloadMultipartForm(writer, entries, codePath, logger)
	}
}
