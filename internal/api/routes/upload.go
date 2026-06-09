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
	"github.com/matteoepitech/flick/internal/api/serverconfig"
	"io"
	"net/http"
	"os"
)

// UploadFileHandler: Build the upload file handler.
//
// Returns:
// - http.HandlerFunc: The handler function.
func UploadFileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		if serverconfig.Conf.MaxFileSizeMb > 0 {
			r.Body = http.MaxBytesReader(w, r.Body, int64(serverconfig.Conf.MaxFileSizeMb)*1024*1024)
		}

		err := r.ParseMultipartForm(100 << 20)
		if err != nil {
			logging.LogInfoError("Cannot parse multipart form: %v", err)
			WriteError(w, http.StatusBadRequest, "Payload too large or invalid request")
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			logging.LogInfoError("Cannot parse uploaded file: %v", err)
			WriteError(w, http.StatusBadRequest, "Cannot parse the file")
			return
		}
		defer file.Close()

		m := new(metadata.Metadata)

		// SetExpiration / SetMaxDownloadCount log the precise reason themselves.
		if !metadata.SetExpiration(m, r.URL.Query().Get("expiration")) {
			WriteError(w, http.StatusBadRequest, "Invalid expiration time")
			return
		}

		if !metadata.SetMaxDownloadCount(m, r.URL.Query().Get("maxDownloadCount")) {
			WriteError(w, http.StatusBadRequest, "Invalid max download count")
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

		if err := os.MkdirAll(path.GetDataDir()+codeDir, 0755); err != nil {
			logging.LogInfoError("Cannot create directory for code %q: %v", codeDir, err)
			WriteError(w, http.StatusInternalServerError, "Cannot save the file")
			return
		}
		metadata.CreateMetadataFile(*m, path.GetDataDir()+codeDir+"/", codeDir)

		dst, err := os.Create(path.GetDataDir() + codeDir + "/" + header.Filename)
		if err != nil {
			logging.LogInfoError("Cannot create destination file %q for code %q: %v", header.Filename, codeDir, err)
			WriteError(w, http.StatusInternalServerError, "Cannot save the file")
			return
		}
		defer dst.Close()

		fileBytes, err := io.Copy(dst, file)
		if err != nil {
			logging.LogInfoError("Cannot write uploaded file %q for code %q: %v", header.Filename, codeDir, err)
			WriteError(w, http.StatusInternalServerError, "Error while copying the file")
			return
		}
		logging.LogInfoSuccess("Received file %q with code %q (%d bytes)", header.Filename, codeDir, fileBytes)
		fmt.Fprintf(w, "%s", codeDir)
		IncUploads()
	}
}
