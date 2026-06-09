/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/download
** File description:
** Download route handler
 */

package routes

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	codepkg "github.com/matteoepitech/flick/internal/api/code"
	"github.com/matteoepitech/flick/internal/api/logging"
	"github.com/matteoepitech/flick/internal/api/metadata"
	"github.com/matteoepitech/flick/internal/api/path"
)

// onDownloadFinished: When a download is done we do this.
//
// Params:
// - code (string): The code to update.
func onDownloadFinished(code string) {
	metadataPath := path.GetDataDir() + "/" + code + "/"
	metadataFile, err := os.Open(metadataPath + "." + code + "-metadata.json")
	if err != nil {
		logging.LogInfoError("Cannot open metadata file for code %q: %v", code, err)
		return
	}
	defer metadataFile.Close()

	var meta metadata.Metadata
	if err := json.NewDecoder(metadataFile).Decode(&meta); err != nil {
		logging.LogInfoError("Cannot decode metadata file for code %q: %v", code, err)
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
func doDownloadMultipartForm(writer *multipart.Writer, entries []os.DirEntry, path string) {
	for _, entry := range entries {
		fullPath := path + "/" + entry.Name()
		file, err := os.Open(fullPath)

		if err != nil {
			logging.LogInfoError("Cannot open file %q: %v", entry.Name(), err)
			continue
		}

		info, err := file.Stat()
		if err != nil {
			logging.LogInfoError("Cannot stat file %q: %v", entry.Name(), err)
			file.Close()
			continue
		}

		part, err := writer.CreateFormFile("file", entry.Name())
		if err != nil {
			logging.LogInfoError("Cannot create multipart part for file %q: %v", entry.Name(), err)
			file.Close()
			continue
		}

		_, err = io.Copy(part, file)
		file.Close()

		if err != nil {
			logging.LogInfoError("Cannot send file %q: %v", entry.Name(), err)
			continue
		}
		logging.LogInfoSuccess("Sent file %q (%d bytes)", entry.Name(), info.Size())
	}
}

// DownloadFileHandler: Build the download file handler.
//
// Returns:
// - http.HandlerFunc: The handler function.
func DownloadFileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			WriteError(w, http.StatusBadRequest, "Missing code parameter")
			return
		}

		if codepkg.IsCodeAlreadyExistInList(code) == false {
			logging.LogInfoError("Code %q is expired or does not exist", code)
			WriteError(w, http.StatusNotFound, "Code not found")
			return
		}

		codePath := path.GetDataDir() + code
		entries, err := os.ReadDir(codePath)
		if err != nil {
			logging.LogInfoError("Cannot read data directory for code %q: %v", code, err)
			WriteError(w, http.StatusInternalServerError, "Cannot read the files")
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

		metadataFilename := "." + code + "-metadata.json"
		var filteredEntries []os.DirEntry
		for _, entry := range entries {
			if entry.Name() != metadataFilename {
				filteredEntries = append(filteredEntries, entry)
			}
		}

		doDownloadMultipartForm(writer, filteredEntries, codePath)
		onDownloadFinished(code)
		IncDownloads()
	}
}
