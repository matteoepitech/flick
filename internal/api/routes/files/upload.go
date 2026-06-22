/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/files/upload
** File description:
** Upload route handler
 */

package files

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matteoepitech/flick/internal/api/code"
	"github.com/matteoepitech/flick/internal/api/database"
	"github.com/matteoepitech/flick/internal/api/logging"
	"github.com/matteoepitech/flick/internal/api/metadata"
	"github.com/matteoepitech/flick/internal/api/path"
	"github.com/matteoepitech/flick/internal/api/quota"
	"github.com/matteoepitech/flick/internal/api/routes"
	"github.com/matteoepitech/flick/internal/api/routes/account"
	"github.com/matteoepitech/flick/internal/api/serverconfig"
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
// - result2 (bool): True when the uploader is an anonymous (not logged-in) user.
// - result3 (bool): True when the uploader is a registered but blocked account.
// - result4 (error): An error if the header is missing, invalid or unknown.
func resolveUploaderID(r *http.Request, queries *database.Queries) (string, bool, bool, error) {
	uploaderID := r.Header.Get("X-Flick-User-ID")
	if uploaderID == "" {
		return "", false, false, fmt.Errorf("missing uploader id")
	}

	var userUUID pgtype.UUID
	if err := userUUID.Scan(uploaderID); err != nil {
		return "", false, false, fmt.Errorf("invalid user id %q: %w", uploaderID, err)
	}

	if _, err := queries.GetAnonymousUserByID(r.Context(), userUUID); err == nil {
		return uploaderID, true, false, nil
	}

	if user, err := queries.GetUserByID(r.Context(), userUUID); err == nil {
		return uploaderID, false, user.Blocked, nil
	}

	return "", false, false, fmt.Errorf("unknown user id %q", uploaderID)
}

// UploadFileHandler: Build the upload file handler. When a `group_id` query
// parameter is present the upload is bound to that group: only a maintainer or
// owner may post it, the transfer becomes private (downloadable only by members
// through the same /download endpoint) and is recorded so the group can list it.
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

		if err := r.ParseMultipartForm(100 << 20); err != nil {
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

		var incoming int64
		for _, header := range headers {
			incoming += header.Size
		}

		m := new(metadata.Metadata)
		m.FileZipSize = incoming

		groupParam := r.URL.Query().Get("group_id")
		var groupID, folderID, uploaderID pgtype.UUID
		var quotaUsed int64
		var quotaLimitMb int
		isGroup := groupParam != ""

		if isGroup {
			if err := groupID.Scan(groupParam); err != nil {
				routes.WriteError(w, http.StatusBadRequest, "Invalid group id")
				return
			}
			caller, status, err := account.RequireGroupMaintainer(r.Context(), queries, account.TokenFromHeader(r), groupID)
			if err != nil {
				routes.WriteError(w, status, err.Error())
				return
			}
			uploaderID = caller.ID

			used, err := quota.UsedByGroupID(path.GetDataDir(), groupParam)
			if err != nil {
				logging.LogInfoError("Cannot read group quota for %q: %v", groupParam, err)
				routes.WriteError(w, http.StatusInternalServerError, "Cannot save the file")
				return
			}
			quotaUsed = used
			quotaLimitMb = serverconfig.Conf.GroupQuotaMb

			if folderParam := r.URL.Query().Get("folder_id"); folderParam != "" {
				if err := folderID.Scan(folderParam); err != nil {
					routes.WriteError(w, http.StatusBadRequest, "Invalid folder id")
					return
				}
				folder, err := queries.GetGroupFolderByID(r.Context(), folderID)
				if err != nil || folder.GroupID != groupID {
					routes.WriteError(w, http.StatusBadRequest, "Folder not found")
					return
				}
			}

			m.MaxDownloadCount = 0
			if !metadata.SetGroupID(m, groupParam) {
				routes.WriteError(w, http.StatusInternalServerError, "Cannot save the file")
				return
			}
		} else {
			rawID, isAnonymous, blocked, err := resolveUploaderID(r, queries)
			if err != nil {
				logging.LogInfoError("Cannot identify uploader: %v", err)
				routes.WriteError(w, http.StatusBadRequest, "Invalid or unknown user")
				return
			}
			if blocked {
				routes.WriteError(w, http.StatusForbidden, "Account blocked")
				return
			}

			used, err := quota.UsedByUploaderID(path.GetDataDir(), rawID)
			if err != nil {
				logging.LogInfoError("Cannot read user quota for %q: %v", rawID, err)
				routes.WriteError(w, http.StatusInternalServerError, "Cannot save the file")
				return
			}
			quotaUsed = used
			quotaLimitMb = serverconfig.Conf.UserQuotaMb
			if isAnonymous {
				quotaLimitMb = serverconfig.Conf.AnonymousQuotaMb
			}

			if !metadata.SetUploaderID(m, rawID) {
				routes.WriteError(w, http.StatusBadRequest, "Invalid or unknown user")
				return
			}
			if !metadata.SetMaxDownloadCount(m, r.URL.Query().Get("maxDownloadCount")) {
				routes.WriteError(w, http.StatusBadRequest, "Invalid max download count")
				return
			}
			if !metadata.SetPassword(m, r.Header.Get("X-Flick-Password")) {
				routes.WriteError(w, http.StatusBadRequest, "Invalid password")
				return
			}
		}

		usedMb := (quotaUsed + incoming) / (1024 * 1024)
		if quotaLimitMb > 0 && usedMb > int64(quotaLimitMb) {
			routes.WriteError(w, http.StatusRequestEntityTooLarge, "Storage quota exceeded")
			return
		}

		// SetExpiration / SetChecksum log the precise reason themselves.
		if !metadata.SetExpiration(m, r.URL.Query().Get("expiration")) {
			routes.WriteError(w, http.StatusBadRequest, "Invalid expiration time")
			return
		}

		if !metadata.SetChecksum(m, r.Header.Get("X-Flick-Checksum")) {
			routes.WriteError(w, http.StatusBadRequest, "Invalid or missing checksum")
			return
		}

		message := ""
		if raw := r.Header.Get("X-Flick-Message"); raw != "" {
			decoded, err := base64.StdEncoding.DecodeString(raw)
			if err != nil {
				logging.LogInfoError("Cannot decode message header: %v", err)
				routes.WriteError(w, http.StatusBadRequest, "Invalid message")
				return
			}
			message = string(decoded)
		}
		if !metadata.SetMessage(m, message) {
			routes.WriteError(w, http.StatusBadRequest, "Invalid message")
			return
		}

		// No error check on this one.
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

		if isGroup {
			if _, err := queries.CreateGroupUpload(r.Context(), database.CreateGroupUploadParams{
				GroupID:    groupID,
				FolderID:   folderID,
				Code:       codeDir,
				UploaderID: uploaderID,
			}); err != nil {
				logging.LogInfoError("Cannot record group upload for code %q: %v", codeDir, err)
				routes.WriteError(w, http.StatusInternalServerError, "Cannot save the group upload")
				return
			}
		}

		fmt.Fprintf(w, "%s", codeDir)
		routes.IncUploads()
	}
}
