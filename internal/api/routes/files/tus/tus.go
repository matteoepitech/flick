/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/files/tus/tus
** File description:
** tus 1.0.0 resumable upload endpoint (receive side)
 */

package tus

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/exp/slog"

	"github.com/tus/tusd/v2/pkg/filestore"
	tusd "github.com/tus/tusd/v2/pkg/handler"

	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/path"
	"github.com/Flick-Corp/flick/internal/api/serverconfig"
)

const (
	BasePath                = "/api/v1/upload/"
	uploadsSubdir           = ".tus-uploads"
	metaKeyResolvedUploader = "flickResolvedUploader"
)

// uploadsDir: Resolve the directory holding the tus in-progress uploads.
//
// Returns:
// - result1 (string): The absolute tus uploads directory.
func uploadsDir() string {
	return filepath.Join(path.GetDataDir(), uploadsSubdir)
}

// NewHandler: Build the tus 1.0.0 upload handler. Chunks are stored on disk,
// validated against the Flick auth and quota at creation time, and finalized into
// a share code once the last chunk lands.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.Handler): The tus handler, to be mounted with http.StripPrefix.
// - result2 (error): An error if the handler could not be built.
func NewHandler(queries *database.Queries) (http.Handler, error) {
	storeDir := uploadsDir()
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create tus uploads directory %q: %w", storeDir, err)
	}

	store := filestore.New(storeDir)
	composer := tusd.NewStoreComposer()
	store.UseIn(composer)

	var maxSize int64
	if serverconfig.Conf.MaxFileSizeMb > 0 {
		maxSize = int64(serverconfig.Conf.MaxFileSizeMb) * 1024 * 1024
	}

	handler, err := tusd.NewHandler(tusd.Config{
		BasePath:                  BasePath,
		StoreComposer:             composer,
		MaxSize:                   maxSize,
		DisableDownload:           true,
		Logger:                    slog.New(slog.NewTextHandler(io.Discard, nil)),
		PreUploadCreateCallback:   preUploadCreate(queries),
		PreFinishResponseCallback: preFinishResponse(queries),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create tus handler: %w", err)
	}

	logging.LogInfoSuccess("tus upload endpoint mounted at %s (max %d MB)", BasePath, serverconfig.Conf.MaxFileSizeMb)
	return handler, nil
}
