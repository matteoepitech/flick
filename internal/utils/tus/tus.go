/*
** FLICK PROJECT, 2026
** flick/internal/utils/tus/tus
** File description:
** Shared tus upload settings used by every Go sender (CLI upload + explorer)
 */

package tus

// ChunkSize is the size of each PATCH chunk streamed to the server during a tus
// upload. Keeping it fixed bounds the sender's memory use (go-tus buffers one
// chunk at a time) no matter how large the archive is. Change it here to adjust
// every Go sender at once; keep it in sync with the web client
// (web/lib/api.ts TUS_CHUNK_SIZE) so all senders behave identically.
const ChunkSize int64 = 16 * 1024 * 1024 // 16 MiB
