/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/files/download_info
** File description:
** Download info route handler (lists a code's contents without consuming it)
 */

package files

import (
	"archive/zip"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	codepkg "github.com/matteoepitech/flick/internal/api/code"
	"github.com/matteoepitech/flick/internal/api/logging"
	"github.com/matteoepitech/flick/internal/api/path"
	"github.com/matteoepitech/flick/internal/api/routes"
)

// downloadInfoItem: one item behind a code.
type downloadInfoItem struct {
	Name      string `json:"name"`
	IsFolder  bool   `json:"isFolder"`
	FileCount int    `json:"fileCount"`
	Size      int64  `json:"size"`
}

// downloadInfoResponse: the listing returned by the info endpoint.
type downloadInfoResponse struct {
	Items []downloadInfoItem `json:"items"`
}

// DownloadInfoHandler: List the files behind a code without consuming a download.
//
// Returns:
// - http.HandlerFunc: The handler function.
func DownloadInfoHandler() http.HandlerFunc {
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

		codePath := path.GetDataDir() + code
		metadataFilename := "." + code + "-metadata.json"

		entries, err := os.ReadDir(codePath)
		if err != nil {
			logging.LogInfoError("Cannot list files for code %q: %v", code, err)
			routes.WriteError(w, http.StatusInternalServerError, "Cannot read the files")
			return
		}

		var resp downloadInfoResponse
		var order []string
		byTop := map[string]*downloadInfoItem{}
		for _, entry := range entries {
			if entry.Name() == metadataFilename {
				continue
			}

			reader, err := zip.OpenReader(codePath + "/" + entry.Name())
			if err != nil {
				logging.LogInfoError("Cannot open archive for code %q: %v", code, err)
				routes.WriteError(w, http.StatusInternalServerError, "Cannot read the files")
				return
			}

			for _, f := range reader.File {
				if f.FileInfo().IsDir() {
					continue
				}

				name := f.Name
				isFolder := false
				if i := strings.Index(name, "/"); i != -1 {
					name = name[:i]
					isFolder = true
				}

				item, ok := byTop[name]
				if !ok {
					item = &downloadInfoItem{Name: name}
					byTop[name] = item
					order = append(order, name)
				}
				if isFolder {
					item.IsFolder = true
				}
				item.FileCount++
				item.Size += int64(f.UncompressedSize64)
			}
			reader.Close()
		}

		for _, name := range order {
			resp.Items = append(resp.Items, *byTop[name])
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			logging.LogInfoError("Cannot encode info response for code %q: %v", code, err)
		}
	}
}
