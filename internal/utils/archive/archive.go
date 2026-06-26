/*
** FLICK PROJECT, 2026
** flick/internal/utils/archive/archive
** File description:
** Shared zip-archiving helpers used to bundle local files/folders into a single
** archive before uploading them to the server (used by the upload command and
** the interactive explorer).
 */

package archive

import (
	"archive/zip"
	"crypto/rand"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// ProgressFunc: Callback invoked during zip creation with the number of bytes
// written so far and the estimated total (sum of input file sizes). Use the zero
// value (nil) to skip progress reporting.
type ProgressFunc func(written, total int64)

// Progress bar writer type structure.
type progressWriter struct {
	w       io.Writer
	written *int64
	fn      ProgressFunc
	total   int64
}

// TotalSize: Walk every src and return the sum of all regular file sizes.
//
// Params:
// - srcs ([]string): The files and/or directories to sum.
//
// Returns:
// - result1 (int64): The total byte size.
// - result2 (error): An error if a path cannot be walked.
func TotalSize(srcs []string) (int64, error) {
	var total int64
	for _, src := range srcs {
		err := filepath.WalkDir(src, func(_ string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if fi, err := d.Info(); err == nil {
				total += fi.Size()
			}
			return nil
		})
		if err != nil {
			return 0, err
		}
	}
	return total, nil
}

// archiveRoot: Choose the base against which src's entries are named inside the
// archive. A local relative path keeps its full structure, so "dir1/a.txt" and
// "dir2/a.txt" stay distinct.
//
// Params:
// - src (string): The file or directory path passed on the command line.
//
// Returns:
// - result1 (string): The base directory to compute archive-relative paths from.
func archiveRoot(src string) string {
	if filepath.IsLocal(src) {
		return "."
	}
	return filepath.Dir(src)
}

// ToTemp: Build a single zip archive of every src into a temporary file
// and return its path. If progress is non-nil it is called on every
// write chunk with the cumulative byte count and the total input size.
//
// Params:
// - srcs ([]string): The files and/or directories to archive together.
// - progress (ProgressFunc): Optional progress callback, may be nil.
//
// Returns:
// - result1 (string): The path to the temporary zip file.
// - result2 (error): An error if occured.
func ToTemp(srcs []string, progress ProgressFunc) (string, error) {
	var total int64
	if progress != nil {
		var err error
		total, err = TotalSize(srcs)
		if err != nil {
			return "", fmt.Errorf("Failure: Cannot calculate total size: %w", err)
		}
	}

	tmp, err := os.CreateTemp("", "flick-upload-*.zip")
	if err != nil {
		return "", fmt.Errorf("Failure: Cannot create temp archive: %w", err)
	}
	defer tmp.Close()

	zw := zip.NewWriter(tmp)
	var written int64
	for _, src := range srcs {
		root := archiveRoot(src)
		if err := addToZip(zw, root, src, &written, progress, total); err != nil {
			zw.Close()
			os.Remove(tmp.Name())
			return "", fmt.Errorf("Failure: Cannot build archive: %w", err)
		}
	}

	if err := zw.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("Failure: Cannot finalize archive: %w", err)
	}
	return tmp.Name(), nil
}

// addToZip: Add file(s) to the zip archive, storing each one with relative path.
//
// Params:
// - zw (*zip.Writer): The zip writer to add entries to.
// - root (string): The base directory used to compute relative paths.
// - path (string): The current file or directory to add.
// - written (*int64): Running total of bytes written, updated after every file.
// - progress (ProgressFunc): Optional progress callback, may be nil.
// - total (int64): Total input size for the progress denominator.
//
// Returns:
// - result1 (error): An error if occured.
func addToZip(zw *zip.Writer, root string, path string, written *int64, progress ProgressFunc, total int64) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := addToZip(zw, root, filepath.Join(path, entry.Name()), written, progress, total); err != nil {
				return err
			}
		}
		return nil
	}

	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}

	w, err := zw.Create(filepath.ToSlash(relPath))
	if err != nil {
		return err
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if progress != nil {
		_, err = io.Copy(&progressWriter{w: w, written: written, fn: progress, total: total}, f)
	} else {
		_, err = io.Copy(w, f)
	}
	return err
}

// Write: Write in the progress bar writter.
//
// Params:
// - p ([]byte): The bytes to write in
//
// Returns:
// - result1 (int): Number of bytes written
// - result2 (error): An error
func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.w.Write(p)
	if n > 0 {
		*pw.written += int64(n)
		pw.fn(*pw.written, pw.total)
	}
	return n, err
}

// RandomName: A random uuid-style name for the uploaded archive.
//
// Returns:
// - result1 (string): The "<uuid>.zip" archive name.
func RandomName() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("flick-%d.zip", time.Now().UnixNano())
	}

	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x.zip", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
