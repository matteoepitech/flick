/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/files/upload
** File description:
** Upload route handler
 */

package files

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matteoepitech/flick/internal/api/code"
	"github.com/matteoepitech/flick/internal/api/database"
	"github.com/matteoepitech/flick/internal/api/logging"
	"github.com/matteoepitech/flick/internal/api/metadata"
	"github.com/matteoepitech/flick/internal/api/path"
	"github.com/matteoepitech/flick/internal/api/routes"
	"github.com/matteoepitech/flick/internal/api/serverconfig"
	"path/filepath"
)

// resolveUploaderID: Validate the mandatory X-Flick-User-ID header against the
// anonymous_users and users tables, and return the uploader UUID. The uploader
// is required: a missing, malformed or unknown id is an error.
//
// Params:
// - r (*http.Request): The upload request.
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (string): The validated uploader UUID.
// - result2 (bool): True when the uploader is a registered but blocked account.
// - result3 (error): An error if the header is missing, invalid or unknown.
func resolveUploaderID(r *http.Request, queries *database.Queries) (string, bool, error) {
	uploaderID := r.Header.Get("X-Flick-User-ID")
	if uploaderID == "" {
		return "", false, fmt.Errorf("missing uploader id")
	}

	var userUUID pgtype.UUID
	if err := userUUID.Scan(uploaderID); err != nil {
		return "", false, fmt.Errorf("invalid user id %q: %w", uploaderID, err)
	}

	if _, err := queries.GetAnonymousUserByID(r.Context(), userUUID); err == nil {
		return uploaderID, false, nil
	}

	if user, err := queries.GetUserByID(r.Context(), userUUID); err == nil {
		return uploaderID, user.Blocked, nil
	}

	return "", false, fmt.Errorf("unknown user id %q", uploaderID)
}

// UploadFileHandler: Build the upload file handler.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - http.HandlerFunc: The handler function.
func UploadFileHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		if serverconfig.Conf.MaxFileSizeMb > 0 {
			r.Body = http.MaxBytesReader(w, r.Body, int64(serverconfig.Conf.MaxFileSizeMb)*1024*1024)
		}

		err := r.ParseMultipartForm(100 << 20)
		if err != nil {
			logging.LogInfoError("Cannot parse multipart form: %v", err)
			routes.WriteError(w, http.StatusBadRequest, "Payload too large or invalid request")
			return
		}

		if r.MultipartForm == nil || len(r.MultipartForm.File["file"]) == 0 {
			logging.LogInfoError("No file part in upload")
			routes.WriteError(w, http.StatusBadRequest, "Cannot parse the file")
			return
		}
		headers := r.MultipartForm.File["file"]

		uploaderID, blocked, err := resolveUploaderID(r, queries)
		if err != nil {
			logging.LogInfoError("Cannot identify uploader: %v", err)
			routes.WriteError(w, http.StatusBadRequest, "Invalid or unknown user")
			return
		}
		if blocked {
			routes.WriteError(w, http.StatusForbidden, "Account blocked")
			return
		}

		m := new(metadata.Metadata)
		if !metadata.SetUploaderID(m, uploaderID) {
			routes.WriteError(w, http.StatusBadRequest, "Invalid or unknown user")
			return
		}

		// SetExpiration / SetMaxDownloadCount / SetChecksum log the precise reason themselves.
		if !metadata.SetExpiration(m, r.URL.Query().Get("expiration")) {
			routes.WriteError(w, http.StatusBadRequest, "Invalid expiration time")
			return
		}

		if !metadata.SetMaxDownloadCount(m, r.URL.Query().Get("maxDownloadCount")) {
			routes.WriteError(w, http.StatusBadRequest, "Invalid max download count")
			return
		}

		if !metadata.SetChecksum(m, r.Header.Get("X-Flick-Checksum")) {
			routes.WriteError(w, http.StatusBadRequest, "Invalid or missing checksum")
			return
		}

		metadata.SetEncrypted(m, r.Header.Get("X-Flick-Encrypted") == "true")

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
			routes.WriteError(w, http.StatusInternalServerError, "Cannot save the file")
			return
		}
		metadata.CreateMetadataFile(*m, path.GetDataDir()+codeDir+"/", codeDir)

		for _, header := range headers {
			name := filepath.Base(header.Filename)

			file, err := header.Open()
			if err != nil {
				logging.LogInfoError("Cannot open uploaded file %q for code %q: %v", name, codeDir, err)
				routes.WriteError(w, http.StatusBadRequest, "Cannot parse the file")
				return
			}

			dst, err := os.Create(path.GetDataDir() + codeDir + "/" + name)
			if err != nil {
				file.Close()
				logging.LogInfoError("Cannot create destination file %q for code %q: %v", name, codeDir, err)
				routes.WriteError(w, http.StatusInternalServerError, "Cannot save the file")
				return
			}

			fileBytes, err := io.Copy(dst, file)
			file.Close()
			dst.Close()
			if err != nil {
				logging.LogInfoError("Cannot write uploaded file %q for code %q: %v", name, codeDir, err)
				routes.WriteError(w, http.StatusInternalServerError, "Error while copying the file")
				return
			}
			logging.LogInfoSuccess("Received file %q with code %q (%d bytes)", name, codeDir, fileBytes)
		}

		fmt.Fprintf(w, "%s", codeDir)
		routes.IncUploads()
	}
}
