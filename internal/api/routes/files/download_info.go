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

	"github.com/Flick-Corp/flick/internal/api/auth"
	codepkg "github.com/Flick-Corp/flick/internal/api/code"
	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/metadata"
	"github.com/Flick-Corp/flick/internal/api/path"
	"github.com/Flick-Corp/flick/internal/api/routes"
	"github.com/jackc/pgx/v5/pgtype"
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
	Items             []downloadInfoItem `json:"items"`
	Encrypted         bool               `json:"encrypted,omitempty"`
	PasswordProtected bool               `json:"passwordProtected,omitempty"`
	Message           string             `json:"message,omitempty"`
}

// DownloadInfoHandler: List the files behind a code without consuming a download.
//
// Params:
//   - queries (*database.Queries): The database queries, used to authorize a
//     member when the code is bound to a group.
//
// Returns:
// - http.HandlerFunc: The handler function.
func DownloadInfoHandler(queries *database.Queries) http.HandlerFunc {
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

		meta, metaErr := metadata.LoadMetadata(path.GetDataDir(), code)

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

		passwordProtected := metaErr == nil && metadata.IsPasswordProtected(&meta)

		message := ""
		if metaErr == nil {
			message = meta.Message
		}

		if passwordProtected && !metadata.CheckPassword(&meta, r.Header.Get("X-Flick-Password")) {
			var total int64
			for _, entry := range entries {
				if entry.Name() == metadataFilename {
					continue
				}
				if info, err := entry.Info(); err == nil {
					total += info.Size()
				}
			}

			resp := downloadInfoResponse{
				PasswordProtected: true,
				Encrypted:         metaErr == nil && meta.Encrypted,
				Message:           message,
				Items:             []downloadInfoItem{{Name: "password protected content", FileCount: 1, Size: total}},
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				logging.LogInfoError("Cannot encode info response for code %q: %v", code, err)
			}
			return
		}

		// Check if this is encrypted or not
		if metaErr == nil && meta.Encrypted {
			var total int64
			for _, entry := range entries {
				if entry.Name() == metadataFilename {
					continue
				}
				if info, err := entry.Info(); err == nil {
					total += info.Size()
				}
			}

			resp := downloadInfoResponse{
				Encrypted: true,
				Message:   message,
				Items:     []downloadInfoItem{{Name: "encrypted content", FileCount: 1, Size: total}},
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				logging.LogInfoError("Cannot encode info response for code %q: %v", code, err)
			}
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

		resp.Message = message
		for _, name := range order {
			resp.Items = append(resp.Items, *byTop[name])
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			logging.LogInfoError("Cannot encode info response for code %q: %v", code, err)
		}
	}
}
