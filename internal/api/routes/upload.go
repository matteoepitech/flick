/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/upload
** File description:
** Upload route handler
 */

package routes

import (
	"fmt"
	"github.com/matteoepitech/flick/internal/api/code"
	"github.com/matteoepitech/flick/internal/api/logging"
	"io"
	"net/http"
	"os"
)

// UploadFileHandler: Build the upload file handler.
//
// Params:
// - dataDir (string): The directory where files are stored.
// - logger (logging.Logger): The logger to use.
//
// Returns:
// - http.HandlerFunc: The handler function.
func UploadFileHandler(dataDir string, logger logging.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "This endpoint is meant to be POST only", http.StatusNotFound)
			return
		}

		r.ParseMultipartForm(100 << 20)

		file, header, err := r.FormFile("file")
		if err != nil {
			logger.InfoError("Error while parsing an uploaded file")
			http.Error(w, "Cannot parse the file", http.StatusBadRequest)
			return
		}
		defer file.Close()
		var codeDir string = code.CodeGen()
		os.MkdirAll(dataDir+codeDir, 0755)
		dst, err := os.Create(dataDir + codeDir + "/" + header.Filename)
		if err != nil {
			logger.InfoError("Error while uploading a file of code <%s>", header.Filename)
			http.Error(w, "Cannot save the file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		fileBytes, err := io.Copy(dst, file)
		if err != nil {
			logger.InfoError("Error while uploading a file of code <%s>", header.Filename)
			http.Error(w, "Error while copying the file", http.StatusInternalServerError)
			return
		}
		logger.InfoSuccess("Received a file with code <%s> (%d bytes)", codeDir, fileBytes)
		fmt.Fprintf(w, "%s\n", codeDir)
	}
}
