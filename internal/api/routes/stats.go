/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/stats
** File description:
** Stats route handler
 */

package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/Flick-Corp/flick/internal/api/code"
	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/metadata"
	"github.com/Flick-Corp/flick/internal/api/path"
)

// Variables that will contain the total uploads/downloads.
var (
	totalUploads   atomic.Uint64
	totalDownloads atomic.Uint64
)

// IncrementStatUploads: Increment the total uploads counter at the Flick instance level.
func IncrementStatUploads() {
	totalUploads.Add(1)
}

// IncrementStatDownloads: Increment the total downloads counter at the Flick instance level.
func IncrementStatDownloads() {
	totalDownloads.Add(1)
}

// TotalStatUploads: Read the total uploads counter on the Flick instance.
//
// Returns:
// - result1 (uint64): Total uploads.
func TotalStatUploads() uint64 {
	return totalUploads.Load()
}

// TotalStatDownloads: Read the total downloads counter on the Flick instance.
//
// Retruns:
// - result1(uint64): Total downloads.
func TotalStatDownloads() uint64 {
	return totalDownloads.Load()
}

// TotalStatUserCount: Read the total of user registered on the Flick instance.
//
// Params:
// - queries (*database.Queries): The query engine.
// - c (context.Context):	The context of the query.
//
// Returns:
// - result1 (int64): Total user (can return -1 if there is any error, no error is returned).
func TotalStatUserCount(queries *database.Queries, c context.Context) int64 {
	userCount, err := queries.CountUsers(c)
	if err != nil {
		userCount = -1
	}
	return userCount
}

// TotalStatStorageUsed: Walk the data directory and sum the size of every stored file at this moment on the Flick instance.
//
// Returns:
// - result1 (int64): Total bytes used by uploaded files at this moment.
func TotalStatStorageUsed() int64 {
	var total int64

	entries, err := os.ReadDir(path.GetDataDir())
	if err != nil {
		return -1
	}
	for _, entry := range entries {
		if entry.IsDir() == false {
			continue
		}

		metadata, err := metadata.LoadMetadata(path.GetDataDir(), entry.Name())
		if err != nil {
			continue
		}

		total += metadata.FileZipSize
	}
	return total
}

// TotalStatActiveCodeCount: Read the total of active code at this moment on the Flick instance.
//
// Returns:
// - result1 (uint64): Total cache active at this moment.
func TotalStatActiveCodeCount() int {
	return code.Cache.ItemCount()
}

// ServerStatsHandler: API endpoint to get informations about the Flick instance.
// Currently returning:
//   - Number of code actives
//   - Total uploads
//   - Total downloads
//   - Number of users registered
//   - Total storage used
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - http.HandlerFunc: The handler function.
func ServerStatsHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		payload := map[string]any{
			"timestamp":      time.Now().UTC().Format(time.RFC3339),
			"activeCodes":    TotalStatActiveCodeCount(),
			"totalUploads":   TotalStatUploads(),
			"totalDownloads": TotalStatDownloads(),
			"userCount":      TotalStatUserCount(queries, r.Context()),
			"storageBytes":   TotalStatStorageUsed(),
		}

		w.Header().Set("Content-Type", "application/json")

		data, err := json.Marshal(payload)
		if err != nil {
			logging.LogInfoError("Cannot encode stats response: %v", err)
			WriteError(w, http.StatusInternalServerError, "Cannot encode response")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}
}
