/*
** FLICK PROJECT, 2026
** flick/internal/api/auth/password
** File description:
** Password functions
 */

package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Flick-Corp/flick/internal/api/logging"
	"golang.org/x/crypto/argon2"
)

// Argon2id parameters, shared by hashing and verification so stored hashes stay verifiable.
const (
	argonTime    uint32 = 1
	argonMemory  uint32 = 64 * 1024
	argonThreads uint8  = 4
	argonKeyLen  uint32 = 32
	saltLen      int    = 16
)

// HashUserPassword: Hash a user's password using salt.
//
// Params:
// - password (string): The password to hash.
//
// Returns:
// - result1 (string): Final password hashed, encoded as "salt$hash".
func HashUserPassword(password string) string {
	salt := make([]byte, saltLen)
	rand.Read(salt)
	if _, err := rand.Read(salt); err != nil {
		logging.LogInfoError("Cannot generate password salt: %v", err)
		return ""
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	return fmt.Sprintf("%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash))
}

// VerifyUserPassword: Check a user's password against a stored "salt$hash" value.
//
// Params:
// - password (string): The password to check.
// - encoded (string): The stored "salt$hash" value.
//
// Returns:
// - result1 (bool): true if the password matches.
func VerifyUserPassword(password string, encoded string) bool {
	parts := strings.Split(encoded, "$")
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

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return subtle.ConstantTimeCompare(hash, expected) == 1
}
