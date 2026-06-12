/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/stats
** File description:
** Stats route handler
 */

package routes

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/matteoepitech/flick/internal/api/code"
	"github.com/matteoepitech/flick/internal/api/database"
	"github.com/matteoepitech/flick/internal/api/path"
)

// Variables that will contain the total uploads/downloads.
var (
	totalUploads   atomic.Uint64
	totalDownloads atomic.Uint64
)

// IncUploads: Increment the total uploads counter.
func IncUploads() {
	totalUploads.Add(1)
}

// IncDownloads: Increment the total downloads counter.
func IncDownloads() {
	totalDownloads.Add(1)
}

// Uploads: Read the total uploads counter.
//
// Returns:
// - result1 (uint64): Total uploads.
func Uploads() uint64 {
	return totalUploads.Load()
}

// Downloads: Read the total downloads counter.
//
// Retruns:
// - result1(uint64): Total downloads.
func Downloads() uint64 {
	return totalDownloads.Load()
}

// storageUsed: Walk the data directory and sum the size of every stored file.
//
// Returns:
// - result1 (int64): Total bytes used by uploaded files.
func storageUsed() int64 {
	var total int64
	_ = filepath.WalkDir(path.GetDataDir(), func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

// SendStats: Build the stats handler.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - http.HandlerFunc: The handler function.
func SendStats(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		userCount, err := queries.CountUsers(r.Context())
		if err != nil {
			userCount = -1
		}

		payload := map[string]any{
			"timestamp":      time.Now().UTC().Format(time.RFC3339),
			"activeCodes":    code.Cache.ItemCount(),
			"totalUploads":   Uploads(),
			"totalDownloads": Downloads(),
			"userCount":      userCount,
			"storageBytes":   storageUsed(),
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}
}
