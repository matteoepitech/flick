/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/files/tus/finalize
** File description:
** Transport-agnostic finalization of a fully-received upload
 */

package tus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Flick-Corp/flick/internal/api/code"
	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/metadata"
	"github.com/Flick-Corp/flick/internal/api/path"
	"github.com/jackc/pgx/v5/pgtype"
)

// uploadInput: The validated, transport-agnostic description of a finished
// upload. It is filled from the tus Upload-Metadata keys and the resolved auth so
// the finalization below never touches HTTP or the tus protocol directly.
type uploadInput struct {
	Filename         string
	Checksum         string
	Encrypted        bool
	Password         string
	Message          string
	Expiration       string
	MaxDownloadCount string
	GroupID          string
	FolderID         string
	ResolvedUploader string
	IsGroup          bool
	Size             int64
}

// buildMetadata: Assemble the on-disk Metadata from a validated upload input,
// reusing the very same setters the legacy multipart handler relied on so the
// stored fields stay identical. A false return means a value was rejected (the
// setter logs the precise reason itself).
//
// Params:
// - in (uploadInput): The validated upload description.
//
// Returns:
// - result1 (*metadata.Metadata): The assembled metadata, or nil on rejection.
// - result2 (bool): True when every field was accepted.
func buildMetadata(in uploadInput) (*metadata.Metadata, bool) {
	m := new(metadata.Metadata)
	m.FileZipSize = in.Size

	if in.IsGroup {
		m.MaxDownloadCount = 0
		if !metadata.SetGroupID(m, in.GroupID) {
			return nil, false
		}
	} else {
		if !metadata.SetUploaderID(m, in.ResolvedUploader) {
			return nil, false
		}
		if !metadata.SetMaxDownloadCount(m, in.MaxDownloadCount) {
			return nil, false
		}
		if !metadata.SetPassword(m, in.Password) {
			return nil, false
		}
	}

	if !metadata.SetExpiration(m, in.Expiration) {
		return nil, false
	}
	if !metadata.SetChecksum(m, in.Checksum) {
		return nil, false
	}
	if !metadata.SetMessage(m, in.Message) {
		return nil, false
	}
	metadata.SetEncrypted(m, in.Encrypted)

	return m, true
}

// finalizeUpload: Turn a fully-received, checksum-verified archive into a live
// Flick transfer. It generates a unique share code, persists the metadata file,
// moves the assembled archive into the code directory and records the group
// binding when needed. This is the single finalization path shared by every
// upload, decoupled from the tus handler that calls it.
//
// On any failure after the code is reserved, the reserved code and its directory
// are rolled back so a failed upload never leaves a dangling code behind.
//
// Params:
// - ctx (context.Context): The request context.
// - queries (*database.Queries): The database queries.
// - in (uploadInput): The validated upload description.
// - assembledPath (string): The path to the fully-received archive on disk.
//
// Returns:
// - result1 (string): The generated share code.
// - result2 (error): An error if the upload could not be finalized.
func finalizeUpload(ctx context.Context, queries *database.Queries, in uploadInput, assembledPath string) (string, error) {
	m, ok := buildMetadata(in)
	if !ok {
		return "", fmt.Errorf("invalid upload metadata")
	}

	// Reserve a unique share code in the RAM cache.
	var shareCode string
	for {
		shareCode = code.CodeGen()
		if code.IsCodeAlreadyExistInList(shareCode) {
			continue
		}
		code.AddCodeToList(shareCode, in.Expiration)
		break
	}

	codeDir := path.GetDataDir() + shareCode
	if err := os.MkdirAll(codeDir, 0755); err != nil {
		_ = code.DeleteCode(shareCode)
		return "", fmt.Errorf("cannot create directory for code %q: %w", shareCode, err)
	}

	metadata.CreateMetadataFile(*m, codeDir+"/", shareCode)

	dst := filepath.Join(codeDir, filepath.Base(in.Filename))
	if err := os.Rename(assembledPath, dst); err != nil {
		_ = code.DeleteCode(shareCode)
		return "", fmt.Errorf("cannot move assembled archive for code %q: %w", shareCode, err)
	}

	if in.IsGroup {
		if err := bindGroupUpload(ctx, queries, in, shareCode); err != nil {
			_ = code.DeleteCode(shareCode)
			return "", err
		}
	}

	logging.LogInfoSuccess("Finalized upload %q with code %q (%d bytes)", filepath.Base(in.Filename), shareCode, in.Size)
	return shareCode, nil
}

// bindGroupUpload: Record a group-bound transfer so the group can list it. The
// uploader, group and folder ids were validated when the upload was created.
//
// Params:
// - ctx (context.Context): The request context.
// - queries (*database.Queries): The database queries.
// - in (uploadInput): The validated upload description.
// - shareCode (string): The generated share code to bind.
//
// Returns:
// - result1 (error): An error if the binding could not be recorded.
func bindGroupUpload(ctx context.Context, queries *database.Queries, in uploadInput, shareCode string) error {
	var groupID, folderID, uploaderID pgtype.UUID
	if err := groupID.Scan(in.GroupID); err != nil {
		return fmt.Errorf("invalid group id %q: %w", in.GroupID, err)
	}
	if in.FolderID != "" {
		if err := folderID.Scan(in.FolderID); err != nil {
			return fmt.Errorf("invalid folder id %q: %w", in.FolderID, err)
		}
	}
	if err := uploaderID.Scan(in.ResolvedUploader); err != nil {
		return fmt.Errorf("invalid uploader id %q: %w", in.ResolvedUploader, err)
	}

	if _, err := queries.CreateGroupUpload(ctx, database.CreateGroupUploadParams{
		GroupID:    groupID,
		FolderID:   folderID,
		Code:       shareCode,
		UploaderID: uploaderID,
	}); err != nil {
		return fmt.Errorf("cannot record group upload for code %q: %w", shareCode, err)
	}
	return nil
}
