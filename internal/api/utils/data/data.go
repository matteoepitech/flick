/*
** FLICK PROJECT, 2026
** flick/internal/api/utils/data
** File description:
** Data utils source file
 */

package data

import (
	"os"
	"path/filepath"

	"github.com/matteoepitech/flick/internal/api/path"
)

// DeleteDataDirWithCode: Delete a directory of data using his code.
//
// Params:
// - code (string): The code to delete.
func DeleteDataDirWithCode(code string) {
	dataDir := filepath.Join(path.GetDataDir(), code)
	subdir, _ := os.ReadDir(dataDir)
	for _, files := range subdir {
		os.Remove(dataDir + files.Name())
	}
	os.Remove(dataDir)
}
