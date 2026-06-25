/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/files/tus/cleanup
** File description:
** Periodic cleanup of abandoned tus uploads
 */

package tus

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Flick-Corp/flick/internal/api/logging"
)

const (
	cleanupInterval = 30 * time.Minute
	uploadMaxAge    = 24 * time.Hour
)

// StartCleanupRoutine: Periodically sweep abandoned tus uploads in the background.
// An incomplete transfer leaves an <id> chunk file and an <id>.info sidecar
// behind; a client that never resumes would otherwise keep them forever. The
// routine runs once at startup and then on a ticker until ctx is cancelled.
//
// Params:
// - ctx (context.Context): Cancels the routine when the server stops.
func StartCleanupRoutine(ctx context.Context) {
	go func() {
		if err := cleanupStaleUploads(uploadsDir(), uploadMaxAge); err != nil {
			logging.LogInfoError("Initial tus cleanup failed: %v", err)
		}

		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := cleanupStaleUploads(uploadsDir(), uploadMaxAge); err != nil {
					logging.LogInfoError("Periodic tus cleanup failed: %v", err)
				}
			}
		}
	}()
}

// cleanupStaleUploads: Remove tus .info sidecars (and their chunk files) whose
// last modification is older than maxAge. tus rewrites the .info file on every
// chunk, so a fresh modtime means the upload is still progressing; anything older
// is abandoned. A finished upload is moved out by the finalization step, so it is
// never seen here.
//
// Params:
// - dir (string): The tus uploads directory to sweep.
// - maxAge (time.Duration): The age past which an upload is considered abandoned.
//
// Returns:
// - result1 (error): An error if the directory cannot be read.
func cleanupStaleUploads(dir string, maxAge time.Duration) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".info") {
			continue
		}

		info, err := entry.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}

		infoPath := filepath.Join(dir, entry.Name())
		binPath := strings.TrimSuffix(infoPath, ".info")
		cleanupArtifacts(binPath, infoPath)
		logging.LogInfoSuccess("Removed stale tus upload %q", strings.TrimSuffix(entry.Name(), ".info"))
	}
	return nil
}

// cleanupArtifacts: Best-effort removal of a tus chunk file and/or its .info
// sidecar. Missing files are ignored; other failures are logged but not fatal.
//
// Params:
// - binPath (string): The chunk file to remove, or "" to skip.
// - infoPath (string): The .info sidecar to remove, or "" to skip.
func cleanupArtifacts(binPath, infoPath string) {
	if binPath != "" {
		if err := os.Remove(binPath); err != nil && !os.IsNotExist(err) {
			logging.LogInfoError("Cannot remove tus artifact %q: %v", binPath, err)
		}
	}
	if infoPath != "" {
		if err := os.Remove(infoPath); err != nil && !os.IsNotExist(err) {
			logging.LogInfoError("Cannot remove tus artifact %q: %v", infoPath, err)
		}
	}
}
