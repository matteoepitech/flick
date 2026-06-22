/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/upload
** File description:
** Upload flick source
 */

package commands

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/matteoepitech/flick/internal/api/utils"
	"github.com/matteoepitech/flick/internal/cli/config"
	"github.com/matteoepitech/flick/internal/cli/network"
	"github.com/matteoepitech/flick/internal/utils/checksum"
	"github.com/matteoepitech/flick/internal/utils/encryption"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/tiagomelo/go-clipboard/clipboard"
)

// uploadItem: a file or folder about to be uploaded.
type uploadItem struct {
	name      string
	isFolder  bool
	fileCount int
	size      int64
}

// quotaResponse: the usage returned by the /quota endpoint.
type quotaResponse struct {
	UsedBytes int64 `json:"usedBytes"`
	LimitMb   int64 `json:"limitMb"`
}

// doUploadRequest: Do the upload request on the server.
//
// Params:
// - req (*http.Request): The request HTTP.
// - exp (string): The expiration of this upload, shown next to the code.
// - keyFragment (string): The base64url encryption key to append to the code.
//
// Returns:
// - result1 (error): An error occured.
func doUploadRequest(req *http.Request, exp string, keyFragment string) error {
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

	shareCode := string(body)
	if keyFragment != "" {
		shareCode += "#" + keyFragment
	}
	fmt.Printf("\nCode: %s "+utils.Yellow+"[%s left]\n"+utils.Reset, shareCode, exp)
	if keyFragment != "" {
		fmt.Println(utils.Dim + "Encrypted: share the whole code, the part after # is the decryption key." + utils.Reset)
	}

	c := clipboard.New(clipboard.ClipboardOptions{Primary: true})
	if err := c.CopyText(shareCode); err != nil {
		return err
	}

	return nil
}

// encryptToTemp: Encrypt the archive at srcPath into a new temporary file under
// key, returning that file's path. The caller is responsible for removing it.
//
// Params:
// - srcPath (string): The plaintext archive to encrypt.
// - key (encryption.Key): The single-use key to encrypt with.
//
// Returns:
// - result1 (string): The path to the temporary ciphertext file.
// - result2 (error): An error if occured.
func encryptToTemp(srcPath string, key encryption.Key) (string, error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("Failure: Cannot open the archive: %w", err)
	}
	defer src.Close()

	tmp, err := os.CreateTemp("", "flick-upload-*.enc")
	if err != nil {
		return "", fmt.Errorf("Failure: Cannot create encrypted temp file: %w", err)
	}

	if err := encryption.Encrypt(tmp, src, key); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", fmt.Errorf("Failure: Cannot encrypt the archive: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("Failure: Cannot finalize the encrypted archive: %w", err)
	}
	return tmp.Name(), nil
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

// statUploadItem: Summarize a local path for the pre-upload listing.
//
// Params:
// - path (string): The file or directory passed on the command line.
//
// Returns:
// - result1 (uploadItem): The name, kind and size of the path.
// - result2 (error): An error if the path cannot be read.
func statUploadItem(path string) (uploadItem, error) {
	info, err := os.Stat(path)
	if err != nil {
		return uploadItem{}, err
	}

	item := uploadItem{name: filepath.Base(path), isFolder: info.IsDir()}
	if !info.IsDir() {
		item.size = info.Size()
		return item, nil
	}

	err = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if fi, err := d.Info(); err == nil {
			item.fileCount++
			item.size += fi.Size()
		}
		return nil
	})
	return item, err
}

// printUploadInfo: Show the files and folders about to be uploaded.
//
// Params:
// - args ([]string): The paths passed on the command line.
func printUploadInfo(args []string) {
	fmt.Println("\nThis upload contains:")
	for _, arg := range args {
		item, err := statUploadItem(arg)
		if err != nil {
			continue
		}
		if item.isFolder {
			fmt.Printf("  • "+utils.Blue+"%s/ (%d files, %s)\n"+utils.Reset, item.name, item.fileCount, humanSize(item.size))
		} else {
			fmt.Printf("  • "+utils.Dim+"%s (%s)\n"+utils.Reset, item.name, humanSize(item.size))
		}
	}
}

// fetchQuota: Read the current storage usage for the uploader.
//
// Params:
// - userID (string): The uploader id sent through the X-Flick-User-ID header.
//
// Returns:
// - result1 (quotaResponse): The used and limit megabytes.
// - result2 (error): An error if the server cannot be reached.
func fetchQuota(userID string) (quotaResponse, error) {
	var q quotaResponse

	req, err := http.NewRequest("GET", config.Conf.APIBaseURL()+"/quota", nil)
	if err != nil {
		return q, err
	}
	req.Header.Set("X-Flick-User-ID", userID)

	resp, err := network.SharedClient.Do(req)
	if err != nil {
		return q, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return q, fmt.Errorf("quota request failed")
	}
	if err := json.NewDecoder(resp.Body).Decode(&q); err != nil {
		return q, err
	}
	return q, nil
}

// printQuotaBar: Draw a textual bar showing how much of the quota is used.
//
// Params:
// - q (quotaResponse): The used and limit megabytes.
func printQuotaBar(q quotaResponse) {
	if q.LimitMb <= 0 {
		fmt.Printf("Quota: %s used (unlimited)\n", humanSize(q.UsedBytes))
		return
	}

	limitBytes := q.LimitMb * 1024 * 1024
	const width = 20
	ratio := float64(q.UsedBytes) / float64(limitBytes)
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * width)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	fmt.Printf("Quota: [%s] %s / %s used (%.0f%%)\n", bar, humanSize(q.UsedBytes), humanSize(limitBytes), ratio*100)
}

// RunUpload: Run the upload command.
//
// Params:
// - cmd (*cobra.Command): The command.
// - args ([]string): The differents arguments of this command.
// - exp (string): The expiration of this upload.
// - mdc (string): The Max Download Count of this upload.
// - encrypt (bool): Encrypt the archive end-to-end before uploading.
// - password (string): Protect the download with a password, or empty for none.
// - message (string): A personal note shown to the downloader, or empty for none.
//
// Returns:
// - result1 (error): An error if occured.
func RunUpload(cmd *cobra.Command, args []string, exp string, mdc string, encrypt bool, password string, message string) error {
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

	printUploadInfo(args)
	if q, err := fetchQuota(creds.UserID); err == nil {
		printQuotaBar(q)
	} else {
		fmt.Printf(utils.BrightRed + "Quota cannot be fetched" + utils.Reset)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", "Upload these files?")
	line, _ := reader.ReadString('\n')
	if answer := strings.ToLower(strings.TrimSpace(line)); answer != "y" && answer != "yes" {
		fmt.Println("Aborted.")
		return nil
	}

	archivePath, err := archiveToTemp(args)
	if err != nil {
		return err
	}
	defer os.Remove(archivePath)

	uploadPath := archivePath
	var keyFragment string
	if encrypt {
		key, err := encryption.NewKey()
		if err != nil {
			return fmt.Errorf("Failure: Cannot generate an encryption key: %w", err)
		}
		encPath, err := encryptToTemp(archivePath, key)
		if err != nil {
			return err
		}
		defer os.Remove(encPath)
		uploadPath = encPath
		keyFragment = encryption.EncodeKey(key)
	}

	archiveChecksum, err := checksum.HashFile(uploadPath)
	if err != nil {
		return fmt.Errorf("Failure: Cannot checksum the archive: %w", err)
	}

	archive, err := os.Open(uploadPath)
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
	req.Header.Set("X-Flick-Checksum", archiveChecksum)
	if keyFragment != "" {
		req.Header.Set("X-Flick-Encrypted", "true")
	}
	if password != "" {
		req.Header.Set("X-Flick-Password", password)
	}
	if message != "" {
		req.Header.Set("X-Flick-Message", base64.StdEncoding.EncodeToString([]byte(message)))
	}
	req.ContentLength = int64(body.Len())

	if err := doUploadRequest(req, exp, keyFragment); err != nil {
		return err
	}
	if password != "" {
		fmt.Println(utils.BrightGreen + "Password protected: the downloader must enter the password to download." + utils.Reset)
	}
	fmt.Println("Code copied to clipboard.")
	return nil
}
