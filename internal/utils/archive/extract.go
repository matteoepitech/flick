/*
** FLICK PROJECT, 2026
** flick/internal/utils/archive/extract
** File description:
** Shared zip-extraction helpers used to unpack a downloaded archive back into
** real files (used by the download command and the interactive explorer).
 */

package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Extract: Unpack a zip archive into dest, returning the unique top-level entry
// names it produced (handy for a "downloaded X" message). It guards against
// zip-slip: any entry resolving outside dest aborts the whole extraction.
//
// Params:
// - zipPath (string): The path to the zip file on disk.
// - dest (string): The destination directory.
//
// Returns:
// - result1 ([]string): The unique top-level names extracted, in archive order.
// - result2 (error): An error if the archive could not be opened or extracted.
func Extract(zipPath string, dest string) ([]string, error) {
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return nil, err
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open the archive: %w", err)
	}
	defer r.Close()

	seen := make(map[string]struct{})
	var topLevel []string
	for _, f := range r.File {
		target := filepath.Join(absDest, f.Name)

		if target != absDest && !strings.HasPrefix(target, absDest+string(os.PathSeparator)) {
			return nil, fmt.Errorf("unsafe path in archive: %q", f.Name)
		}

		if top := topLevelName(f.Name); top != "" {
			if _, ok := seen[top]; !ok {
				seen[top] = struct{}{}
				topLevel = append(topLevel, top)
			}
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return nil, err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return nil, err
		}
		if err := writeZipEntry(f, target); err != nil {
			return nil, err
		}
	}
	return topLevel, nil
}

// writeZipEntry: Copy a single zip entry to target on disk.
//
// Params:
// - f (*zip.File): The zip entry to extract.
// - target (string): The destination path on disk.
//
// Returns:
// - result1 (error): An error if occured.
func writeZipEntry(f *zip.File, target string) error {
	src, err := f.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

// topLevelName: Return the first path segment of an archive entry name, i.e. the
// loose file or the top folder it belongs to ("a/b/c.txt" -> "a", "p.png" -> "p.png").
//
// Params:
// - name (string): The archive-relative entry name.
//
// Returns:
// - result1 (string): The top-level segment, or "" when the name is empty.
func topLevelName(name string) string {
	name = strings.TrimPrefix(name, "/")
	if before, _, found := strings.Cut(name, "/"); found {
		return before
	}
	return name
}
