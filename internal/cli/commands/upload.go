/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/upload
** File description:
** Upload flick source
 */

package commands

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/matteoepitech/flick/internal/cli/config"
	"github.com/matteoepitech/flick/internal/cli/network"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/tiagomelo/go-clipboard/clipboard"
)

// doUploadRequest: Do the upload request on the server.
//
// Params:
// - req (*http.Request): The request HTTP.
// - exp (string): The expiration of this upload, shown next to the code.
//
// Returns:
// - result1 (error): An error occured.
func doUploadRequest(req *http.Request, exp string) error {
	resp, err := network.SharedClient.Do(req)
	if err != nil {
		return fmt.Errorf("Failure: Cannot access the server: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Failure: Invalid response from the server")
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("Failure: %s", serverErrorMessage(body, resp.Status))
	}

	bodyString := string(body)
	fmt.Printf("\nCode: %s \033[33m[%s left]\033[0m\n", bodyString, exp)

	c := clipboard.New(clipboard.ClipboardOptions{Primary: true})
	if err := c.CopyText(bodyString); err != nil {
		return err
	}

	return nil
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

// archiveToTemp: Build a single zip archive of every src into a temporary file
// and return its path.
//
// Params:
// - srcs ([]string): The files and/or directories to archive together.
//
// Returns:
// - result1 (string): The path to the temporary zip file.
// - result2 (error): An error if occured.
func archiveToTemp(srcs []string) (string, error) {
	tmp, err := os.CreateTemp("", "flick-upload-*.zip")
	if err != nil {
		return "", fmt.Errorf("Failure: Cannot create temp archive: %w", err)
	}
	defer tmp.Close()

	zw := zip.NewWriter(tmp)
	for _, src := range srcs {
		root := archiveRoot(src)
		if err := addToZip(zw, root, src); err != nil {
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
//
// Returns:
// - result1 (error): An error if occured.
func addToZip(zw *zip.Writer, root string, path string) error {
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
			if err := addToZip(zw, root, filepath.Join(path, entry.Name())); err != nil {
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

	_, err = io.Copy(w, f)
	return err
}

// randomArchiveName: A random uuid-style name for the uploaded archive.
//
// Returns:
// - result1 (string): The "<uuid>.zip" archive name.
func randomArchiveName() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("flick-%d.zip", time.Now().UnixNano())
	}

	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x.zip", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// RunUpload: Run the upload command.
//
// Params:
// - cmd (*cobra.Command): The command.
// - args ([]string): The differents arguments of this command.
// - exp (string): The expiration of this upload.
// - mdc (string): The Max Download Count of this upload.
//
// Returns:
// - result1 (error): An error if occured.
func RunUpload(cmd *cobra.Command, args []string, exp string, mdc string) error {
	if len(args) < 1 {
		return fmt.Errorf("Failure: Internal CLI error.")
	}

	// Validate every path up front so we fail before touching the network.
	for _, arg := range args {
		if _, err := os.Stat(arg); err != nil {
			return fmt.Errorf("Failure: Cannot get that file: %s", arg)
		}
	}

	serverLimits, err := config.GetServerLimits()
	if err != nil {
		return fmt.Errorf("Failure: Cannot get server limits: %w", err)
	}

	expValue := exp
	if expValue == "" {
		expValue = config.Conf.DefExpTime
	}

	mdcValue := mdc
	if mdcValue == "" {
		mdcValue = strconv.FormatInt(int64(config.Conf.DefDownloadCount), 10)
	}

	if serverLimits.MaxDownloadCount > 0 {
		mdvInt, err := strconv.ParseInt(mdcValue, 10, 32)
		if err == nil && mdvInt > int64(serverLimits.MaxDownloadCount) {
			return fmt.Errorf("Failure: The max download count is too large. The server only allows up to %d downloads.", serverLimits.MaxDownloadCount)
		}
	}

	if serverLimits.MaxExpiration != "" && expValue != "" {
		if serverDuration, err := time.ParseDuration(serverLimits.MaxExpiration); err == nil {
			if clientDuration, err := time.ParseDuration(expValue); err == nil {
				if clientDuration > serverDuration {
					return fmt.Errorf("Failure: The expiration is too large. The server only allows up to %s.", serverLimits.MaxExpiration)
				}
			}
		}
	}
	creds, err := config.EnsureCredentials()
	if err != nil {
		return fmt.Errorf("Failure: Cannot identify on the server: %w", err)
	}

	archivePath, err := archiveToTemp(args)
	if err != nil {
		return err
	}
	defer os.Remove(archivePath)

	archive, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("Failure: Cannot open the archive: %w", err)
	}
	defer archive.Close()

	archiveStat, err := archive.Stat()
	if err != nil {
		return fmt.Errorf("Failure: Cannot stat the archive: %w", err)
	}

	if serverLimits.MaxFileSizeMb > 0 && archiveStat.Size() > int64(serverLimits.MaxFileSizeMb)*1024*1024 {
		return fmt.Errorf("Failure: The upload is too large. The server only accepts up to %d MB.", serverLimits.MaxFileSizeMb)
	}

	label := filepath.Base(args[0])
	if len(args) > 1 {
		label = fmt.Sprintf("%d items", len(args))
	}
	fmt.Printf("Uploading %s... (%d bytes archived)\n", label, archiveStat.Size())

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", randomArchiveName())
	if err != nil {
		return fmt.Errorf("Failure: Cannot create the form file: %w", err)
	}

	if _, err := io.Copy(part, archive); err != nil {
		return fmt.Errorf("Failure: Cannot copy the archive.")
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("Failure: Cannot finalize the upload body: %w", err)
	}

	bar := progressbar.DefaultBytes(int64(body.Len()), "Uploading")
	progressBody := io.TeeReader(body, bar)

	params := url.Values{}
	params.Set("expiration", expValue)
	params.Set("maxDownloadCount", mdcValue)

	reqURL := fmt.Sprintf("%s/upload?%s", config.Conf.APIBaseURL(), params.Encode())

	req, err := http.NewRequest("POST", reqURL, progressBody)
	if err != nil {
		return fmt.Errorf("Failure: Cannot create the request for the server.")
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Flick-User-ID", creds.UserID)
	req.ContentLength = int64(body.Len())

	if err := doUploadRequest(req, exp); err != nil {
		return err
	}
	fmt.Println("Code copied to clipboard.")
	return nil
}
