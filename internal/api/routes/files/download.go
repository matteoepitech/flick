/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/files/download
** File description:
** Download route handler
 */

package files

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	"github.com/Flick-Corp/flick/internal/api/auth"
	codepkg "github.com/Flick-Corp/flick/internal/api/code"
	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/metadata"
	"github.com/Flick-Corp/flick/internal/api/path"
	"github.com/Flick-Corp/flick/internal/api/routes"
	"github.com/jackc/pgx/v5/pgtype"
)

// onDownloadFinished: When a download is done we do this.
//
// Params:
// - code (string): The code to update.
func onDownloadFinished(code string) {
	metadataPath := path.GetDataDir() + "/" + code + "/"
	meta, err := metadata.LoadMetadata(path.GetDataDir(), code)
	if err != nil {
		logging.LogInfoError("Cannot load metadata file for code %q: %v", code, err)
		return
	}

	// WARN: Should we use something to prevent race condition here? Maybe in future.
	if meta.CurrentDownloadCount+1 >= meta.MaxDownloadCount && meta.MaxDownloadCount != 0 {
		if err := codepkg.DeleteCode(code); err != nil {
			logging.LogInfoError("Failed to delete code \"%s\": %v", code, err)
		}
		return
	}
	meta.CurrentDownloadCount += 1
	metadata.CreateMetadataFile(meta, metadataPath, code)
}

// doDownloadMultipartForm: Do the download request.
//
// Params:
// - writer (*multipart.Writer): The writer of the multipart.
// - entries ([]os.DirEntry): The different entries of the directory.
// - path (string): The path.
//
// Returns:
// - error: An error if one occurred.
func doDownloadMultipartForm(writer *multipart.Writer, entries []os.DirEntry, path string) error {
	for _, entry := range entries {
		fullPath := path + "/" + entry.Name()
		file, err := os.Open(fullPath)
		if err != nil {
			logging.LogInfoError("Cannot open file %q: %v", entry.Name(), err)
			return err
		}

		info, err := file.Stat()
		if err != nil {
			logging.LogInfoError("Cannot stat file %q: %v", entry.Name(), err)
			file.Close()
			return err
		}

		part, err := writer.CreateFormFile("file", entry.Name())
		if err != nil {
			logging.LogInfoError("Cannot create multipart part for file %q: %v", entry.Name(), err)
			file.Close()
			return err
		}

		_, err = io.Copy(part, file)
		file.Close()

		if err != nil {
			logging.LogInfoError("Cannot send file %q: %v", entry.Name(), err)
			return err
		}
		logging.LogInfoSuccess("Sent file %q (%d bytes)", entry.Name(), info.Size())
	}
	return nil
}

// DownloadFileHandler: Build the download file handler.
//
// Params:
//   - queries (*database.Queries): The database queries, used to authorize a
//     member when the code is bound to a group.
//
// Returns:
// - http.HandlerFunc: The handler function.
func DownloadFileHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			routes.WriteError(w, http.StatusBadRequest, "Missing code parameter")
			return
		}

		if codepkg.IsCodeAlreadyExistInList(code) == false {
			logging.LogInfoError("Code %q is expired or does not exist", code)
			routes.WriteError(w, http.StatusNotFound, "Code not found")
			return
		}

		meta, metaErr := metadata.LoadMetadata(path.GetDataDir(), code)
		if metaErr != nil {
			logging.LogInfoError("Cannot load metadata for code %q: %v", code, metaErr)
		}

		if metaErr == nil && metadata.IsGroupBound(&meta) {
			var groupID pgtype.UUID
			if err := groupID.Scan(meta.GroupID); err != nil {
				routes.WriteError(w, http.StatusNotFound, "Code not found")
				return
			}
			if _, _, err := auth.RequireGroupMember(r.Context(), queries, auth.GetTokenFromHTTPRequest(r), groupID); err != nil {
				routes.WriteError(w, http.StatusNotFound, "Code not found")
				return
			}
		}

		if metaErr == nil && !metadata.CheckPassword(&meta, r.Header.Get("X-Flick-Password")) {
			routes.WriteError(w, http.StatusUnauthorized, "Invalid or missing password")
			return
		}

		codePath := path.GetDataDir() + code
		entries, err := os.ReadDir(codePath)
		if err != nil {
			logging.LogInfoError("Cannot read data directory for code %q: %v", code, err)
			routes.WriteError(w, http.StatusInternalServerError, "Cannot read the files")
			return
		}

		metadataFilename := "." + code + "-metadata.json"
		var filteredEntries []os.DirEntry
		for _, entry := range entries {
			if entry.Name() != metadataFilename {
				filteredEntries = append(filteredEntries, entry)
			}
		}

		// Only count the files actually transmitted: announcing more than we
		// send leaves client progress bars stuck below 100%.
		var totalSize int64
		for _, entry := range filteredEntries {
			if info, err := entry.Info(); err == nil {
				totalSize += info.Size()
			}
		}

		writer := multipart.NewWriter(w)

		w.Header().Set("Content-Type", writer.FormDataContentType())
		w.Header().Set("X-Total-Size", strconv.FormatInt(totalSize, 10))

		if metaErr == nil && meta.Checksum != "" {
			w.Header().Set("X-Flick-Checksum", meta.Checksum)
		}

		err = doDownloadMultipartForm(writer, filteredEntries, codePath)
		if err != nil {
			writer.Close()
			routes.WriteError(w, http.StatusInternalServerError, "Error transmitting the files")
			return
		}

		if err := writer.Close(); err != nil {
			logging.LogInfoError("Failed to close multipart writer: %v", err)
		}
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		onDownloadFinished(code)
		routes.IncrementStatDownloads()
	}
}
