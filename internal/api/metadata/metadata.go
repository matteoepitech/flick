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
	"strconv"
	"strings"
	"time"

	"github.com/matteoepitech/flick/internal/api/logging"
	"github.com/matteoepitech/flick/internal/api/serverconfig"
	"github.com/matteoepitech/flick/internal/api/utils"
	"github.com/matteoepitech/flick/internal/api/utils/data"
	"github.com/matteoepitech/flick/internal/utils/checksum"
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
	metadataPath := dataDir + code + "/." + code + "-metadata.json"

	file, err := os.Open(metadataPath)
	if err != nil {
		return meta, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&meta)
	return meta, err
}

// createMetadataFile: Creates the metadata file containing the expiration date.
//
// Params:
// - metadata (Metadata): The metadata informations.
// - filepath (string): The filepath to the metadata location.
// - code (string): The generated share code.
func CreateMetadataFile(metadata Metadata, filepath string, code string) {
	metadataPath := filepath + "." + code + "-metadata.json"

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
// - metadata (*Metadata): The metadata to set the expiration.
// - exp (string): The duration of the expiration.
//
// Returns:
// - result1 (bool): Return true if the metadata has been changed, else false.
func SetExpiration(metadata *Metadata, exp string) bool {
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
	if !checkConfigTime(duration) {
		logging.LogInfoError("Expiration time %q exceeds the maximum allowed by configuration", exp)
		return false
	}
	metadata.Expiration = duration.Format(time.RFC3339)
	return true
}

// SetMaxDownloadCount: Defines the max download count based on the received pattern.
//
// Params:
// - metadata (*Metadata): The metadata to modify.
// - maxDownloadCount (string): The max download count string.
//
// Returns:
// - result1 (bool): Return true if the metadata has been changed, else false.
func SetMaxDownloadCount(metadata *Metadata, maxDownloadCount string) bool {
	mdc, err := strconv.Atoi(maxDownloadCount)
	if err != nil {
		logging.LogInfoError("Cannot parse max download count %q: %v", maxDownloadCount, err)
		return false
	}

	if mdc > serverconfig.Conf.MaxDownloadCount {
		logging.LogInfoError("Max download count %q exceeds the maximum allowed by configuration (%d)", maxDownloadCount, serverconfig.Conf.MaxDownloadCount)
		return false
	}

	metadata.MaxDownloadCount = int32(mdc)
	return true
}

// SetUploaderID: Defines the uploader id. The uploader is mandatory, so an empty
// id is rejected. The id is expected to be already validated against the
// database by the caller.
//
// Params:
// - metadata (*Metadata): The metadata to modify.
// - uploaderID (string): The validated uploader UUID.
//
// Returns:
// - result1 (bool): Return true if the metadata has been changed, else false.
func SetUploaderID(metadata *Metadata, uploaderID string) bool {
	if uploaderID == "" {
		logging.LogInfoError("Uploader id is required")
		return false
	}

	metadata.UploaderID = uploaderID
	return true
}

// SetGroupID: Binds the code to a group, making it private (downloadable only
// through the group routes by its members, never by the public code endpoint).
// The id is expected to be already validated against the database by the caller.
//
// Params:
// - metadata (*Metadata): The metadata to modify.
// - groupID (string): The validated group UUID.
//
// Returns:
// - result1 (bool): Return true if the metadata has been changed, else false.
func SetGroupID(metadata *Metadata, groupID string) bool {
	if groupID == "" {
		logging.LogInfoError("Group id is required")
		return false
	}

	metadata.GroupID = groupID
	return true
}

// IsGroupBound: Report whether a code is private to a group, and therefore must
// not be served by the public download endpoint.
//
// Params:
// - metadata (*Metadata): The metadata to inspect.
//
// Returns:
// - result1 (bool): True when the code belongs to a group.
func IsGroupBound(metadata *Metadata) bool {
	return metadata.GroupID != ""
}

// SetChecksum: Defines the BLAKE3 checksum of the uploaded archive, as computed
// and sent by the client. The checksum lets the downloader confirm the bytes it
// receives are intact. A missing or malformed digest is rejected.
//
// Params:
// - metadata (*Metadata): The metadata to modify.
// - sum (string): The hex-encoded BLAKE3 digest sent by the client.
//
// Returns:
// - result1 (bool): Return true if the metadata has been changed, else false.
func SetChecksum(metadata *Metadata, sum string) bool {
	if !checksum.IsValidHex(sum) {
		logging.LogInfoError("Invalid or missing checksum %q", sum)
		return false
	}

	metadata.Checksum = sum
	return true
}

// SetMessage: Attach an optional personal note the uploader wants the downloader
// to see.
//
// Params:
// - metadata (*Metadata): The metadata to modify.
// - message (string): The note chosen by the uploader, or empty for none.
//
// Returns:
// - result1 (bool): Return true if the message is acceptable, else false.
func SetMessage(metadata *Metadata, message string) bool {
	message = strings.TrimSpace(message)
	if len(message) > maxMessageLen {
		logging.LogInfoError("Message is too long (%d > %d)", len(message), maxMessageLen)
		return false
	}

	metadata.Message = message
	return true
}

// SetEncrypted: Record whether the uploaded archive is end-to-end encrypted, as
// declared by the client through the X-Flick-Encrypted header.
//
// Params:
// - metadata (*Metadata): The metadata to modify.
// - encrypted (bool): True when the client encrypted the archive before upload.
func SetEncrypted(metadata *Metadata, encrypted bool) {
	metadata.Encrypted = encrypted
}

// SetPassword: Protect the code with a download password. The plaintext password
// is hashed with Argon2id and only the "salt$hash" is stored, so the server
// never keeps the password itself. An empty password leaves the code public.
//
// Params:
// - metadata (*Metadata): The metadata to modify.
// - password (string): The plaintext password chosen by the uploader, or empty.
func SetPassword(metadata *Metadata, password string) bool {
	if password == "" {
		return true
	}

	salt := make([]byte, pwSaltLen)
	if _, err := rand.Read(salt); err != nil {
		logging.LogInfoError("Cannot generate password salt: %v", err)
		return false
	}
	hash := argon2.IDKey([]byte(password), salt, pwArgonTime, pwArgonMemory, pwArgonThreads, pwArgonKeyLen)
	metadata.PasswordHash = fmt.Sprintf("%s$%s",
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash))
	return true
}

// IsPasswordProtected: Report whether a download password guards this code.
//
// Params:
// - metadata (*Metadata): The metadata to inspect.
//
// Returns:
// - result1 (bool): True when a password must be supplied to download.
func IsPasswordProtected(metadata *Metadata) bool {
	return metadata.PasswordHash != ""
}

// CheckPassword: Verify a candidate password against the stored Argon2id hash. A
// code with no password always passes, so callers can use this as the single
// access gate regardless of whether protection is enabled.
//
// Params:
// - metadata (*Metadata): The metadata holding the stored hash.
// - password (string): The candidate password supplied by the downloader.
//
// Returns:
// - result1 (bool): True when access is granted.
func CheckPassword(metadata *Metadata, password string) bool {
	if metadata.PasswordHash == "" {
		return true
	}

	parts := strings.Split(metadata.PasswordHash, "$")
	if len(parts) != 2 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	hash := argon2.IDKey([]byte(password), salt, pwArgonTime, pwArgonMemory, pwArgonThreads, pwArgonKeyLen)
	return subtle.ConstantTimeCompare(hash, expected) == 1
}

// CheckExpirationToRemove: Will check and remove every expired files/folders.
//
// Params:
// - dataDir (string): The data directory.
//
// Returns:
// - result1 (error): An error if occured.
func CheckExpirationToRemove(dataDir string) error {
	content, err := os.ReadDir(dataDir)
	if err != nil {
		return err
	}

	for _, entries := range content {
		if entries.IsDir() {
			code := entries.Name()
			file, err := os.Open(dataDir + code + "/." + code + "-metadata.json")
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
				data.DeleteDataDirWithCode(entries.Name())
			}
		}
	}
	return nil
}

// checkConfigTime: Checks that the duration set by the user is in config bounds.
//
// Params:
// - duration (time.Time): The duration passed by the user.
//
// Returns:
// - result1 (bool): True if in config bounds, else false.
func checkConfigTime(duration time.Time) bool {
	maxExp, err := utils.ParseExpirationTime(serverconfig.Conf.MaxExpiration)
	if err != nil {
		logging.LogInfoError("Cannot parse max expiration time %q from configuration: %v", serverconfig.Conf.MaxExpiration, err)
		return false
	}

	return duration.Before(maxExp.Add(time.Second))
}
