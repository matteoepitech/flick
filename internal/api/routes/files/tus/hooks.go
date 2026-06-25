/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/files/tus/hooks
** File description:
** tus lifecycle hooks: authorize on create, verify and finalize on finish
 */

package tus

import (
	"maps"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/tus/tusd/v2/pkg/filestore"
	tusd "github.com/tus/tusd/v2/pkg/handler"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/path"
	"github.com/Flick-Corp/flick/internal/api/quota"
	"github.com/Flick-Corp/flick/internal/api/routes"
	"github.com/Flick-Corp/flick/internal/api/routes/account"
	"github.com/Flick-Corp/flick/internal/api/routes/files"
	"github.com/Flick-Corp/flick/internal/api/serverconfig"
	"github.com/Flick-Corp/flick/internal/utils/checksum"
)

// preUploadCreate: tus hook invoked before a new upload is created. It validates
// the Flick auth, the quota and the required metadata up front, then stashes the
// resolved uploader id into the upload metadata so the finalization step can run
// without re-reading the (possibly absent) auth headers on the final chunk.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1: The tus PreUploadCreateCallback.
func preUploadCreate(queries *database.Queries) func(tusd.HookEvent) (tusd.HTTPResponse, tusd.FileInfoChanges, error) {
	return func(hook tusd.HookEvent) (tusd.HTTPResponse, tusd.FileInfoChanges, error) {
		in, err := authorizeUpload(hook, queries)
		if err != nil {
			return tusd.HTTPResponse{}, tusd.FileInfoChanges{}, err
		}

		newMeta := make(tusd.MetaData, len(hook.Upload.MetaData)+1)
		maps.Copy(newMeta, hook.Upload.MetaData)
		newMeta[metaKeyResolvedUploader] = in.ResolvedUploader

		return tusd.HTTPResponse{}, tusd.FileInfoChanges{MetaData: newMeta}, nil
	}
}

// authorizeUpload: Validate the Flick metadata, auth and quota for a new tus
// upload, returning the partially-filled upload input with the uploader resolved.
// Every rejection is a tus Error carrying the proper HTTP status.
//
// Params:
// - hook (tusd.HookEvent): The tus creation hook event.
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (uploadInput): The validated upload input with ResolvedUploader set.
// - result2 (error): A tus Error on rejection, or nil when accepted.
func authorizeUpload(hook tusd.HookEvent, queries *database.Queries) (uploadInput, error) {
	in := readUploadInput(hook.Upload.MetaData)
	in.Size = hook.Upload.Size

	if hook.Upload.SizeIsDeferred || in.Size <= 0 {
		return in, tusd.NewError("ERR_FLICK_SIZE", "Upload length is required", http.StatusBadRequest)
	}
	if serverconfig.Conf.MaxFileSizeMb > 0 && in.Size > int64(serverconfig.Conf.MaxFileSizeMb)*1024*1024 {
		return in, tusd.NewError("ERR_FLICK_TOO_LARGE", "File exceeds the maximum allowed size", http.StatusRequestEntityTooLarge)
	}
	if in.Filename == "" {
		return in, tusd.NewError("ERR_FLICK_FILENAME", "Missing filename metadata", http.StatusBadRequest)
	}
	if !checksum.IsValidHex(in.Checksum) {
		return in, tusd.NewError("ERR_FLICK_CHECKSUM", "Invalid or missing checksum metadata", http.StatusBadRequest)
	}
	if in.Expiration == "" {
		return in, tusd.NewError("ERR_FLICK_EXPIRATION", "Missing expiration metadata", http.StatusBadRequest)
	}

	ctx := hook.Context

	var quotaUsed int64
	var quotaLimitMb int

	if in.IsGroup {
		var groupID pgtype.UUID
		if err := groupID.Scan(in.GroupID); err != nil {
			return in, tusd.NewError("ERR_FLICK_GROUP", "Invalid group id", http.StatusBadRequest)
		}
		caller, status, err := account.RequireGroupMaintainer(ctx, queries, bearerFromHeader(hook.HTTPRequest.Header), groupID)
		if err != nil {
			return in, tusd.NewError("ERR_FLICK_AUTH", err.Error(), status)
		}
		in.ResolvedUploader = caller.ID.String()

		if in.FolderID != "" {
			var folderID pgtype.UUID
			if err := folderID.Scan(in.FolderID); err != nil {
				return in, tusd.NewError("ERR_FLICK_FOLDER", "Invalid folder id", http.StatusBadRequest)
			}
			folder, err := queries.GetGroupFolderByID(ctx, folderID)
			if err != nil || folder.GroupID != groupID {
				return in, tusd.NewError("ERR_FLICK_FOLDER", "Folder not found", http.StatusBadRequest)
			}
		}

		used, err := quota.UsedByGroupID(path.GetDataDir(), in.GroupID)
		if err != nil {
			return in, tusd.NewError("ERR_FLICK_QUOTA", "Cannot read group quota", http.StatusInternalServerError)
		}
		quotaUsed = used
		quotaLimitMb = serverconfig.Conf.GroupQuotaMb
	} else {
		rawID, isAnonymous, blocked, err := files.ResolveUploaderByID(ctx, queries, hook.HTTPRequest.Header.Get("X-Flick-User-ID"))
		if err != nil {
			return in, tusd.NewError("ERR_FLICK_USER", "Invalid or unknown user", http.StatusBadRequest)
		}
		if blocked {
			return in, tusd.NewError("ERR_FLICK_BLOCKED", "Account blocked", http.StatusForbidden)
		}
		if in.MaxDownloadCount == "" {
			return in, tusd.NewError("ERR_FLICK_MAXDL", "Missing max download count metadata", http.StatusBadRequest)
		}
		in.ResolvedUploader = rawID

		used, err := quota.UsedByUploaderID(path.GetDataDir(), rawID)
		if err != nil {
			return in, tusd.NewError("ERR_FLICK_QUOTA", "Cannot read user quota", http.StatusInternalServerError)
		}
		quotaUsed = used
		quotaLimitMb = serverconfig.Conf.UserQuotaMb
		if isAnonymous {
			quotaLimitMb = serverconfig.Conf.AnonymousQuotaMb
		}
	}

	usedMb := (quotaUsed + in.Size) / (1024 * 1024)
	if quotaLimitMb > 0 && usedMb > int64(quotaLimitMb) {
		return in, tusd.NewError("ERR_FLICK_QUOTA", "Storage quota exceeded", http.StatusRequestEntityTooLarge)
	}

	return in, nil
}

// preFinishResponse: tus hook invoked synchronously once the last chunk lands but
// before the 204 is written. It verifies the assembled archive against the
// announced BLAKE3 checksum, finalizes it into a share code, records the
// id -> code result and cleans up the tus .info sidecar.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1: The tus PreFinishResponseCallback.
func preFinishResponse(queries *database.Queries) func(tusd.HookEvent) (tusd.HTTPResponse, error) {
	return func(hook tusd.HookEvent) (tusd.HTTPResponse, error) {
		assembledPath := hook.Upload.Storage[filestore.StorageKeyPath]
		if assembledPath == "" {
			assembledPath = filepath.Join(uploadsDir(), hook.Upload.ID)
		}
		infoPath := hook.Upload.Storage[filestore.StorageKeyInfoPath]
		if infoPath == "" {
			infoPath = filepath.Join(uploadsDir(), hook.Upload.ID+".info")
		}

		in := readUploadInput(hook.Upload.MetaData)
		in.Size = hook.Upload.Size
		in.ResolvedUploader = hook.Upload.MetaData[metaKeyResolvedUploader]

		// Verify the assembled archive matches the checksum the client announced.
		actual, err := checksum.HashFile(assembledPath)
		if err != nil {
			logging.LogInfoError("Cannot hash assembled tus upload %q: %v", hook.Upload.ID, err)
			cleanupArtifacts(assembledPath, infoPath)
			return tusd.HTTPResponse{}, tusd.NewError("ERR_FLICK_HASH", "Cannot verify upload integrity", http.StatusInternalServerError)
		}
		if !checksum.Equal(actual, in.Checksum) {
			logging.LogInfoError("Checksum mismatch for tus upload %q: got %s want %s", hook.Upload.ID, actual, in.Checksum)
			cleanupArtifacts(assembledPath, infoPath)
			return tusd.HTTPResponse{}, tusd.NewError("ERR_FLICK_CHECKSUM", "Checksum mismatch", http.StatusBadRequest)
		}

		shareCode, err := finalizeUpload(hook.Context, queries, in, assembledPath)
		if err != nil {
			logging.LogInfoError("Cannot finalize tus upload %q: %v", hook.Upload.ID, err)
			cleanupArtifacts(assembledPath, infoPath)
			return tusd.HTTPResponse{}, tusd.NewError("ERR_FLICK_FINALIZE", "Cannot finalize upload", http.StatusInternalServerError)
		}

		// Make the code retrievable by the follow-up GET, then drop the now-stale
		// .info sidecar (finalizeUpload already moved the archive out of the store).
		rememberResult(hook.Upload.ID, shareCode)
		cleanupArtifacts("", infoPath)
		routes.IncUploads()

		return tusd.HTTPResponse{}, nil
	}
}

// readUploadInput: Decode the Flick Upload-Metadata keys (already base64-decoded
// by tus) into an uploadInput. Auth, quota and the assembled size are filled by
// the caller.
//
// Params:
// - meta (tusd.MetaData): The decoded tus Upload-Metadata map.
//
// Returns:
// - result1 (uploadInput): The decoded upload input.
func readUploadInput(meta tusd.MetaData) uploadInput {
	in := uploadInput{
		Filename:         meta["filename"],
		Checksum:         meta["checksum"],
		Encrypted:        meta["encrypted"] == "true",
		Password:         meta["password"],
		Message:          meta["message"],
		Expiration:       meta["expiration"],
		MaxDownloadCount: meta["maxDownloadCount"],
		GroupID:          meta["groupId"],
		FolderID:         meta["folderId"],
	}
	in.IsGroup = in.GroupID != ""
	return in
}

// bearerFromHeader: Extract the bearer token from a raw http.Header. The tus hooks
// expose the request headers rather than the *http.Request that
// account.TokenFromHeader expects, so this mirrors the same extraction logic.
//
// Params:
// - h (http.Header): The incoming request headers.
//
// Returns:
// - result1 (string): The bearer token, or "" when absent or malformed.
func bearerFromHeader(h http.Header) string {
	const prefix = "Bearer "

	header := h.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
