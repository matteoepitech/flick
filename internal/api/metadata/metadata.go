/*
** FLICK PROJECT, 2026
** flick/internal/api/metadata/metadata.go
** File description:
** metadata.go
 */

package metadata

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/path"
	"github.com/Flick-Corp/flick/internal/api/serverconfig"
	"github.com/Flick-Corp/flick/internal/api/utils"
	"github.com/Flick-Corp/flick/internal/utils/checksum"
	"golang.org/x/crypto/argon2"
)

// Argon2id parameters for the share-code password, mirroring the account
// password hashing so stored "salt$hash" values stay consistent across Flick.
const (
	pwArgonTime    uint32 = 1
	pwArgonMemory  uint32 = 64 * 1024
	pwArgonThreads uint8  = 4
	pwArgonKeyLen  uint32 = 32
	pwSaltLen      int    = 16
)

// struct used for the JSON template
type Metadata struct {
	Expiration           string `json:"expiration"`
	CurrentDownloadCount int32  `json:"current_download_count"`
	MaxDownloadCount     int32  `json:"max_download_count"`
	UploaderID           string `json:"uploader_id,omitempty"`
	Checksum             string `json:"checksum,omitempty"`
	Encrypted            bool   `json:"encrypted,omitempty"`
	PasswordHash         string `json:"password_hash,omitempty"`
	Message              string `json:"message,omitempty"`
	GroupID              string `json:"group_id,omitempty"`
	FileZipSize          int64  `json:"file_zip_size"`
}

// maxMessageLen for the message of the code.
const maxMessageLen int = 500

// GetMetadataJSONPath: Build the on-disk path of a code's metadata JSON file.
//
// Params:
// - code (string): The share code whose metadata path to build.
//
// Returns:
// - result1 (string): The full path to the code's metadata JSON file.
func GetMetadataJSONPath(code string) string {
	return path.GetDataDir() + code + "/." + code + "-metadata.json"
}

// LoadMetadata: Read and decode the metadata file of a given code.
//
// Params:
// - dataDir (string): The data directory holding the code folders.
// - code (string): The code whose metadata to load.
//
// Returns:
// - result1 (Metadata): The decoded metadata.
// - result2 (error): An error if occured.
func LoadMetadata(dataDir string, code string) (Metadata, error) {
	var meta Metadata

	file, err := os.Open(GetMetadataJSONPath(code))
	if err != nil {
		return meta, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&meta)
	return meta, err
}

// CreateMetadataFile: Marshal the given metadata and persist it on disk as the
// code's metadata JSON file, at the path derived from the share code.
//
// Params:
// - metadata (Metadata): The metadata informations.
// - code (string): The generated share code.
func CreateMetadataFile(metadata Metadata, code string) {
	metadataPath := GetMetadataJSONPath(code)

	data, err := json.Marshal(metadata)
	if err != nil {
		logging.LogInfoError("Cannot marshal metadata for code %q: %v", code, err)
		return
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		logging.LogInfoError("Cannot write metadata file %q: %v", metadataPath, err)
		return
	}

	logging.LogInfoSuccess("Created metadata file %q", metadataPath)
}

// SetExpiration: Defines the expiration date based on the received pattern.
//
// Params:
// - exp (string): The duration of the expiration.
//
// Returns:
// - result1 (bool): Return true if the metadata has been changed, else false.
func (m *Metadata) SetExpiration(exp string) bool {
	duration, err := utils.ParseExpirationTime(exp)

	if err != nil {
		logging.LogInfoError("Cannot parse expiration time %q: %v", exp, err)
		return false
	}

	if duration.IsZero() {
		logging.LogInfoError("Expiration time %q cannot be zero", exp)
		return false
	}
	if !duration.After(time.Now()) {
		logging.LogInfoError("Expiration time %q is in the past", exp)
		return false
	}

	maxExp, err := utils.ParseExpirationTime(serverconfig.Conf.MaxExpiration)
	if err != nil {
		logging.LogInfoError("Cannot parse max expiration time %q from configuration: %v", serverconfig.Conf.MaxExpiration, err)
		return false
	}
	if !duration.Before(maxExp.Add(time.Second)) {
		logging.LogInfoError("Expiration time %q exceeds the maximum allowed by configuration", exp)
		return false
	}

	m.Expiration = duration.Format(time.RFC3339)
	return true
}

// SetMaxDownloadCount: Defines the max download count based on the received pattern.
//
// Params:
// - maxDownloadCount (string): The max download count string.
//
// Returns:
// - result1 (bool): Return true if the metadata has been changed, else false.
func (m *Metadata) SetMaxDownloadCount(maxDownloadCount string) bool {
	mdc, err := strconv.Atoi(maxDownloadCount)
	if err != nil {
		logging.LogInfoError("Cannot parse max download count %q: %v", maxDownloadCount, err)
		return false
	}

	if mdc < 1 {
		logging.LogInfoError("Max download count %q is invalid: must be at least 1", maxDownloadCount)
		return false
	}

	if !serverconfig.Conf.AllowMultipleDownloads && mdc > 1 {
		logging.LogInfoError("Max download count %q exceeds the maximum allowed by configuration: multiple downloads are disabled", maxDownloadCount)
		return false
	}

	if mdc > serverconfig.Conf.MaxDownloadCount {
		logging.LogInfoError("Max download count %q exceeds the maximum allowed by configuration (%d)", maxDownloadCount, serverconfig.Conf.MaxDownloadCount)
		return false
	}

	m.MaxDownloadCount = int32(mdc)
	return true
}

// SetUploaderID: Defines the uploader id. The uploader is mandatory, so an empty
// id is rejected. The id is expected to be already validated against the
// database by the caller.
//
// Params:
// - uploaderID (string): The validated uploader UUID.
//
// Returns:
// - result1 (bool): Return true if the metadata has been changed, else false.
func (m *Metadata) SetUploaderID(uploaderID string) bool {
	if uploaderID == "" {
		logging.LogInfoError("Uploader id is required")
		return false
	}

	m.UploaderID = uploaderID
	return true
}

// SetGroupID: Binds the code to a group, making it private (downloadable only
// through the group routes by its members, never by the public code endpoint).
// The id is expected to be already validated against the database by the caller.
//
// Params:
// - groupID (string): The validated group UUID.
//
// Returns:
// - result1 (bool): Return true if the metadata has been changed, else false.
func (m *Metadata) SetGroupID(groupID string) bool {
	if groupID == "" {
		logging.LogInfoError("Group id is required")
		return false
	}

	m.GroupID = groupID
	return true
}

// SetChecksum: Defines the BLAKE3 checksum of the uploaded archive, as computed
// and sent by the client. The checksum lets the downloader confirm the bytes it
// receives are intact. A missing or malformed digest is rejected.
//
// Params:
// - sum (string): The hex-encoded BLAKE3 digest sent by the client.
//
// Returns:
// - result1 (bool): Return true if the metadata has been changed, else false.
func (m *Metadata) SetChecksum(sum string) bool {
	if !checksum.IsValidHex(sum) {
		logging.LogInfoError("Invalid or missing checksum %q", sum)
		return false
	}

	m.Checksum = sum
	return true
}

// SetMessage: Attach an optional personal note the uploader wants the downloader
// to see.
//
// Params:
// - message (string): The note chosen by the uploader, or empty for none.
//
// Returns:
// - result1 (bool): Return true if the message is acceptable, else false.
func (m *Metadata) SetMessage(message string) bool {
	message = strings.TrimSpace(message)
	if len(message) > maxMessageLen {
		logging.LogInfoError("Message is too long (%d > %d)", len(message), maxMessageLen)
		return false
	}

	m.Message = message
	return true
}

// SetEncrypted: Record whether the uploaded archive is end-to-end encrypted, as
// declared by the client through the X-Flick-Encrypted header.
//
// Params:
// - encrypted (bool): True when the client encrypted the archive before upload.
//
// Returns:
// - result1 (bool): Return true if the metadata has been changed, else false.
func (m *Metadata) SetEncrypted(encrypted bool) bool {
	m.Encrypted = encrypted
	return true
}

// SetPassword: Protect the code with a download password. The plaintext password
// is hashed with Argon2id and only the "salt$hash" is stored, so the server
// never keeps the password itself. This must only be called when a non-empty
// password was supplied.
//
// Params:
// - password (string): The plaintext password chosen by the uploader.
//
// Returns:
// - result1 (bool): Return true if the metadata has been changed, else false.
func (m *Metadata) SetPassword(password string) bool {
	salt := make([]byte, pwSaltLen)
	if _, err := rand.Read(salt); err != nil {
		logging.LogInfoError("Cannot generate password salt: %v", err)
		return false
	}

	hash := argon2.IDKey([]byte(password), salt, pwArgonTime, pwArgonMemory, pwArgonThreads, pwArgonKeyLen)
	m.PasswordHash = fmt.Sprintf("%s$%s",
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash))
	return true
}

// CheckExpirationToRemove: Walk every code folder in the data directory, decode
// each code's metadata file and delete the folders whose expiration date is in the past.
//
// Returns:
// - result1 (error): An error if occured.
func CheckExpirationToRemove() error {
	dataDir := path.GetDataDir()
	content, err := os.ReadDir(dataDir)
	if err != nil {
		return err
	}

	for _, entry := range content {
		if entry.IsDir() {
			code := entry.Name()
			file, err := os.Open(GetMetadataJSONPath(code))
			if err != nil {
				continue
			}
			defer file.Close()

			var meta Metadata
			err = json.NewDecoder(file).Decode(&meta)

			if err != nil {
				continue
			}

			dateExp, err := (time.Parse(time.RFC3339, meta.Expiration))
			if err != nil {
				continue
			}
			if time.Now().After(dateExp) {
				os.RemoveAll(filepath.Join(dataDir, code))
			}
		}
	}
	return nil
}

// IsGroupCode: Return if the code is private to a group.
//
// Returns:
// - result1 (bool): True when the code belongs to a group.
func (m *Metadata) IsGroupCode() bool {
	return m.GroupID != ""
}

// IsPasswordProtected: Report whether a download password guards this code.
//
// Returns:
// - result1 (bool): True when a password must be supplied to download.
func (m *Metadata) IsPasswordProtected() bool {
	return m.PasswordHash != ""
}

// VerifyCodePassword: Verify a candidate password against the stored Argon2id
// hash. A code with no password always passes, so callers can use this as the
// single access gate regardless of whether protection is enabled.
//
// Params:
// - password (string): The candidate password supplied by the downloader.
//
// Returns:
// - result1 (bool): True when access is granted.
func (m *Metadata) VerifyCodePassword(password string) bool {
	if m.PasswordHash == "" {
		return true
	}

	parts := strings.Split(m.PasswordHash, "$")
	if len(parts) != 2 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	hash := argon2.IDKey([]byte(password), salt, pwArgonTime, pwArgonMemory, pwArgonThreads, pwArgonKeyLen)
	return subtle.ConstantTimeCompare(hash, expectedHash) == 1
}
