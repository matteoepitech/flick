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
	"github.com/matteoepitech/flick/internal/api/metadata"
	"github.com/matteoepitech/flick/internal/api/path"
	"io"
	"net/http"
	"os"
)

// UploadFileHandler: Build the upload file handler.
//
// Params:
// - logger (logging.Logger): The logger to use.
//
// Returns:
// - http.HandlerFunc: The handler function.
func UploadFileHandler(logger logging.Logger) http.HandlerFunc {
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

		m := new(metadata.Metadata)

		if !metadata.SetExpiration(m, r.URL.Query().Get("expiration"), logger) {
			logger.InfoError("Error in expiration time")
			return
		}

		if !metadata.SetMaxDownloadCount(m, r.URL.Query().Get("maxDownloadCount"), logger) {
			logger.InfoError("Error in max download count")
			return
		}

		// Generate a code until we found one correct.
		var codeDir string
		for {
			codeDir = code.CodeGen()
			if code.IsCodeAlreadyExistInList(codeDir) {
				continue
			}
			// Add the code to the cache in RAM to prevent re-use of this code in the future.
			code.AddCodeToList(codeDir, r.URL.Query().Get("expiration"))
			break
		}

		os.MkdirAll(path.GetDataDir()+codeDir, 0755)
		metadata.CreateMetadataFile(*m, path.GetDataDir()+codeDir+"/", codeDir, logger)

		dst, err := os.Create(path.GetDataDir() + codeDir + "/" + header.Filename)
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
		fmt.Fprintf(w, "%s", codeDir)
	}
}
