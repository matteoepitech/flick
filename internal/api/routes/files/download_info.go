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

		if metaErr != nil {
			routes.WriteError(w, http.StatusNotFound, "Code not found")
			return
		}

		if meta.IsGroupCode() {
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

		password := r.Header.Get("X-Flick-Password")
		if meta.IsPasswordProtected() && !meta.VerifyCodePassword(password) {
			resp := downloadInfoResponse{
				PasswordProtected: true,
				Encrypted:         meta.Encrypted,
				Message:           meta.Message,
				Items:             []downloadInfoItem{},
			}

			data, err := json.Marshal(resp)
			if err != nil {
				logging.LogInfoError("Cannot encode stats response: %v", err)
				routes.WriteError(w, http.StatusInternalServerError, "Cannot encode response")
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
			return
		}

		// Check if this is encrypted or not
		if meta.Encrypted {
			resp := downloadInfoResponse{
				Encrypted: true,
				Message:   meta.Message,
				Items:     []downloadInfoItem{},
			}

			data, err := json.Marshal(resp)
			if err != nil {
				logging.LogInfoError("Cannot encode stats response: %v", err)
				routes.WriteError(w, http.StatusInternalServerError, "Cannot encode response")
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
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

		resp.Message = meta.Message
		for _, name := range order {
			resp.Items = append(resp.Items, *byTop[name])
		}

		data, err := json.Marshal(resp)
		if err != nil {
			logging.LogInfoError("Cannot encode stats response: %v", err)
			routes.WriteError(w, http.StatusInternalServerError, "Cannot encode response")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}
