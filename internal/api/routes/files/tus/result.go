/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/files/tus/result
** File description:
** Share-code lookup for finished tus uploads
 */

package tus

import (
	"fmt"
	"net/http"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/Flick-Corp/flick/internal/api/routes"
)

// results maps a finished tus upload id to its assigned share code. The tus
// protocol's final response has no body, so the clients (CLI and web) fetch the
// code in a short follow-up GET keyed by the upload id. The mapping only needs to
// outlive that immediate follow-up, so a short TTL is plenty.
var results = cache.New(30*time.Minute, 10*time.Minute)

// rememberResult: Record the share code assigned to a finished tus upload so the
// follow-up result request can return it.
//
// Params:
// - uploadID (string): The tus upload id.
// - shareCode (string): The share code finalized for that upload.
func rememberResult(uploadID string, shareCode string) {
	results.Set(uploadID, shareCode, cache.DefaultExpiration)
}

// ResultHandler: Serve the share code assigned to a finished tus upload. The
// clients call GET /api/v1/upload-result?id=<uploadId> right after the upload
// completes; the id is the opaque, unguessable tus upload id, so it doubles as
// the access token for its own result.
//
// Returns:
// - http.HandlerFunc: The handler function.
func ResultHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		id := r.URL.Query().Get("id")
		if id == "" {
			routes.WriteError(w, http.StatusBadRequest, "Missing upload id")
			return
		}

		value, ok := results.Get(id)
		shareCode, _ := value.(string)
		if !ok || shareCode == "" {
			routes.WriteError(w, http.StatusNotFound, "Upload not found")
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, shareCode)
	}
}
