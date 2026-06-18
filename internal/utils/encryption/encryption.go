/*
** FLICK PROJECT, 2026
** flick/internal/utils/encryption/encryption
** File description:
** Encryption source file
 */

package encryption

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/secretbox"
)

// KeySize is the length of a secretbox key in bytes (256-bit).
const KeySize = 32

// nonceSize is the length of a secretbox nonce in bytes (192-bit).
const nonceSize = 24

// chunkSize is the amount of plaintext sealed into a single chunk.
const chunkSize = 64 * 1024

// encChunkSize is the size of a full sealed chunk on the wire.
const encChunkSize = chunkSize + secretbox.Overhead

// Key is a single-use symmetric key.
type Key [KeySize]byte

// NewKey: Generate a fresh random key from the OS CSPRNG.
//
// Returns:
// - result1 (Key): The generated key.
// - result2 (error): An error if the system randomness could not be read.
func NewKey() (Key, error) {
	var key Key
	if _, err := rand.Read(key[:]); err != nil {
		return key, fmt.Errorf("cannot generate key: %w", err)
	}
	return key, nil
}

// EncodeKey: Encode a key as a URL-safe, unpadded base64 string suitable for
// appending to a share code.
//
// Params:
// - key (Key): The key to encode.
//
// Returns:
// - result1 (string): The base64url-encoded key.
func EncodeKey(key Key) string {
	return base64.RawURLEncoding.EncodeToString(key[:])
}

// DecodeKey: Decode a base64url-encoded key produced by EncodeKey.
//
// Params:
// - s (string): The encoded key.
//
// Returns:
// - result1 (Key): The decoded key.
// - result2 (error): An error if the string is not a valid key.
func DecodeKey(s string) (Key, error) {
	var key Key
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return key, fmt.Errorf("invalid key encoding: %w", err)
	}
	if len(raw) != KeySize {
		return key, fmt.Errorf("invalid key length: got %d bytes, want %d", len(raw), KeySize)
	}
	copy(key[:], raw)
	return key, nil
}

// makeNonce: Derive the nonce for a given chunk. The 64-bit counter guarantees
// each chunk within a file gets a distinct nonce, and the "last" flag binds the
// position of the final chunk into the authentication tag so a truncated stream
// fails to open.
//
// Params:
// - counter (uint64): The zero-based chunk index.
// - last (bool): Whether this is the final chunk of the stream.
//
// Returns:
// - result1 ([nonceSize]byte): The derived nonce.
func makeNonce(counter uint64, last bool) [nonceSize]byte {
	var nonce [nonceSize]byte
	binary.BigEndian.PutUint64(nonce[0:8], counter)
	if last {
		nonce[8] = 1
	}
	return nonce
}

// Encrypt: Seal everything read from src into dst, chunk by chunk, under key.
// The output is a sequence of secretbox-sealed chunks.
//
// Params:
// - dst (io.Writer): Where the ciphertext is written.
// - src (io.Reader): The plaintext to encrypt.
// - key (Key): The single-use encryption key.
//
// Returns:
// - result1 (error): An error if reading, sealing or writing fails.
func Encrypt(dst io.Writer, src io.Reader, key Key) error {
	secret := [KeySize]byte(key)
	buf := make([]byte, chunkSize)
	var counter uint64

	for {
		n, err := io.ReadFull(src, buf)
		last := false
		switch err {
		case nil:
		case io.EOF:
			last = true
			n = 0
		case io.ErrUnexpectedEOF:
			last = true
		default:
			return fmt.Errorf("cannot read plaintext: %w", err)
		}

		nonce := makeNonce(counter, last)
		sealed := secretbox.Seal(nil, buf[:n], &nonce, &secret)
		if _, err := dst.Write(sealed); err != nil {
			return fmt.Errorf("cannot write ciphertext: %w", err)
		}

		counter++
		if last {
			return nil
		}
	}
}

// Decrypt: Reverse Encrypt, writing the recovered plaintext to dst. Every chunk
// is authenticated; a wrong key, tampered bytes or a truncated stream all
// surface as an error rather than silent corruption.
//
// Params:
// - dst (io.Writer): Where the plaintext is written.
// - src (io.Reader): The ciphertext to decrypt.
// - key (Key): The encryption key recovered from the share code.
//
// Returns:
// - result1 (error): An error if reading, opening or writing fails.
func Decrypt(dst io.Writer, src io.Reader, key Key) error {
	secret := [KeySize]byte(key)
	buf := make([]byte, encChunkSize)
	var counter uint64

	for {
		n, err := io.ReadFull(src, buf)
		last := false
		switch err {
		case nil:
		case io.ErrUnexpectedEOF:
			last = true
		case io.EOF:
			return fmt.Errorf("ciphertext is truncated or corrupted")
		default:
			return fmt.Errorf("cannot read ciphertext: %w", err)
		}

		nonce := makeNonce(counter, last)
		plain, ok := secretbox.Open(nil, buf[:n], &nonce, &secret)
		if !ok {
			return fmt.Errorf("decryption failed: wrong key or corrupted data")
		}
		if _, err := dst.Write(plain); err != nil {
			return fmt.Errorf("cannot write plaintext: %w", err)
		}

		counter++
		if last {
			return nil
		}
	}
}
