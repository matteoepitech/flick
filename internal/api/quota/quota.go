/*
** FLICK PROJECT, 2026
** flick/internal/api/quota/quota
** File description:
** Per-owner upload storage quota
 */

package quota

import (
	"os"

	"github.com/Flick-Corp/flick/internal/api/metadata"
	"github.com/Flick-Corp/flick/internal/api/path"
)

// CalculateQuotaByGroupID: Walk the data directory and sum the stored size of every active
// transfer bound to the given group. Reading the on-disk metadata keeps the
// total correct as expired transfers disappear.
//
// Params:
// - groupID (string): The group UUID owning the transfers.
//
// Returns:
// - result1 (int64): The total bytes used by the group.
// - result2 (error): An error if the data directory cannot be read.
func CalculateQuotaByGroupID(groupID string) (int64, error) {
	return walkSum(path.GetDataDir(), "", groupID)
}

// CalculateQuotaByUploaderID: Walk the data directory and sum the stored size of an
// uploader's active personal transfers. Group transfers count against their
// group, not the uploader, so they are excluded here.
//
// Params:
// - uploaderID (string): The uploader UUID owning the transfers.
//
// Returns:
// - result1 (int64): The total bytes used by the uploader.
// - result2 (error): An error if the data directory cannot be read.
func CalculateQuotaByUploaderID(uploaderID string) (int64, error) {
	return walkSum(path.GetDataDir(), uploaderID, "")
}

// walkSum: Sum the stored size of transfers owned by either an uploader or a
// group. Exactly one of uploaderID / groupID is set, the other is empty.
//
// Params:
// - dataDir (string): The data directory holding the code folders.
// - uploaderID (string): The uploader UUID, or empty for a group lookup.
// - groupID (string): The group UUID, or empty for an uploader lookup.
//
// Returns:
// - result1 (int64): The total bytes matching the owner.
// - result2 (error): An error if the data directory cannot be read.
func walkSum(dataDir string, uploaderID string, groupID string) (int64, error) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return 0, err
	}

	var total int64
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		meta, err := metadata.LoadMetadata(dataDir, entry.Name())
		if err != nil {
			continue
		}
		if groupID != "" {
			if meta.GroupID == groupID {
				total += meta.FileZipSize
			}
		} else if meta.GroupID == "" && meta.UploaderID == uploaderID {
			total += meta.FileZipSize
		}
	}
	return total, nil
}
