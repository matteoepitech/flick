/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/upload
** File description:
** Upload flick source
 */

package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Flick-Corp/flick/internal/api/utils"
	"github.com/Flick-Corp/flick/internal/cli/config"
	"github.com/Flick-Corp/flick/internal/cli/network"
	archiveutil "github.com/Flick-Corp/flick/internal/utils/archive"
	"github.com/Flick-Corp/flick/internal/utils/checksum"
	"github.com/Flick-Corp/flick/internal/utils/encryption"
	tus "github.com/eventials/go-tus"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/tiagomelo/go-clipboard/clipboard"
)

// tusChunkSize is the size of each PATCH chunk streamed to the server. Keeping
// it fixed bounds the client's memory use (go-tus buffers one chunk at a time)
// no matter how large the archive is, and matches the web client's chunk size
// so both senders behave identically.
const tusChunkSize int64 = 16 * 1024 * 1024 // 16 MiB

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

// printShareCode: Show the final share code to the user and copy it to the
// clipboard, appending the encryption key fragment when the upload is encrypted.
//
// Params:
// - code (string): The bare share code returned by the server.
// - exp (string): The expiration of this upload, shown next to the code.
// - keyFragment (string): The base64url encryption key to append to the code.
//
// Returns:
// - result1 (error): An error if the clipboard cannot be written.
func printShareCode(code string, exp string, keyFragment string) error {
	shareCode := code
	if keyFragment != "" {
		shareCode += "#" + keyFragment
	}
	fmt.Printf("\nCode: %s "+utils.Yellow+"[%s left]\n"+utils.Reset, shareCode, exp)
	if keyFragment != "" {
		fmt.Println(utils.Dim + "Encrypted: share the whole code, the part after # is the decryption key." + utils.Reset)
	}

	c := clipboard.New(clipboard.ClipboardOptions{Primary: true})
	return c.CopyText(shareCode)
}

// uploadViaTus: Stream the archive to the server with the tus resumable upload
// protocol. The file is sent in fixed-size chunks so the client's memory use
// stays bounded regardless of the archive size, and a dropped connection can be
// resumed instead of restarting from zero. All the per-upload settings travel as
// tus metadata (the server reads them back when finalizing the transfer); the
// uploader identity travels as the usual X-Flick-User-ID header.
//
// Params:
// - archive (*os.File): The (possibly encrypted) archive to upload, positioned anywhere.
// - size (int64): The exact byte size of the archive.
// - userID (string): The uploader id sent through the X-Flick-User-ID header.
// - archiveChecksum (string): The BLAKE3 hex digest of the archive bytes.
// - encrypted (bool): Whether the archive is end-to-end encrypted.
// - password (string): An optional download password, or empty for none.
// - message (string): An optional personal note, or empty for none.
// - expiration (string): The resolved expiration duration (e.g. "24h").
// - maxDownloadCount (string): The resolved maximum download count.
//
// Returns:
// - result1 (string): The bare share code assigned by the server.
// - result2 (error): An error if the upload or code lookup failed.
func uploadViaTus(archive *os.File, size int64, userID string, archiveChecksum string,
	encrypted bool, password string, message string, expiration string, maxDownloadCount string) (string, error) {

	metadata := tus.Metadata{
		"filename":         archiveutil.RandomName(),
		"checksum":         archiveChecksum,
		"encrypted":        strconv.FormatBool(encrypted),
		"expiration":       expiration,
		"maxDownloadCount": maxDownloadCount,
	}
	if password != "" {
		metadata["password"] = password
	}
	if message != "" {
		metadata["message"] = message
	}

	header := http.Header{}
	header.Set("X-Flick-User-ID", userID)

	client, err := tus.NewClient(config.Conf.APIBaseURL()+"/upload/", &tus.Config{
		ChunkSize:  tusChunkSize,
		HttpClient: network.SharedClient,
		Header:     header,
	})
	if err != nil {
		return "", fmt.Errorf("Failure: Cannot create the upload client: %w", err)
	}

	upload := tus.NewUpload(archive, size, metadata, "")
	uploader, err := client.CreateUpload(upload)
	if err != nil {
		return "", fmt.Errorf("Failure: Cannot start the upload: %w", err)
	}

	bar := progressbar.DefaultBytes(size, "Uploading")
	progress := make(chan tus.Upload)
	uploader.NotifyUploadProgress(progress)
	go func() {
		for u := range progress {
			_ = bar.Set64(u.Offset())
		}
	}()

	if err := uploader.Upload(); err != nil {
		return "", fmt.Errorf("Failure: The upload failed: %w", err)
	}
	_ = bar.Set64(size)
	_ = bar.Finish()

	return fetchUploadCode(uploader.Url(), userID)
}

// fetchUploadCode: Resolve the share code the server assigned to a finished tus
// upload. The tus protocol's final response carries no body, so the code is
// fetched in a short follow-up request keyed by the upload id (the last segment
// of the upload URL). The server has already finalized the transfer by the time
// the upload completes, so this always succeeds immediately.
//
// Params:
// - uploadURL (string): The tus upload URL returned by the server on creation.
// - userID (string): The uploader id sent through the X-Flick-User-ID header.
//
// Returns:
// - result1 (string): The bare share code.
// - result2 (error): An error if the server cannot be reached or refuses.
func fetchUploadCode(uploadURL string, userID string) (string, error) {
	parsed, err := url.Parse(uploadURL)
	if err != nil {
		return "", fmt.Errorf("Failure: Invalid upload URL from the server: %w", err)
	}

	query := url.Values{}
	query.Set("id", path.Base(parsed.Path))
	req, err := http.NewRequest("GET", config.Conf.APIBaseURL()+"/upload-result?"+query.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("Failure: Cannot create the request for the server.")
	}
	req.Header.Set("X-Flick-User-ID", userID)

	resp, err := network.SharedClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failure: Cannot access the server: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Failure: Invalid response from the server")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("Failure: %s", serverErrorMessage(body, resp.Status))
	}

	return strings.TrimSpace(string(body)), nil
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

	archivePath, err := archiveutil.ToTemp(args)
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

	shareCode, err := uploadViaTus(archive, archiveStat.Size(), creds.UserID, archiveChecksum,
		keyFragment != "", password, message, expValue, mdcValue)
	if err != nil {
		return err
	}

	if err := printShareCode(shareCode, expValue, keyFragment); err != nil {
		return err
	}
	if password != "" {
		fmt.Println(utils.BrightGreen + "Password protected: the downloader must enter the password to download." + utils.Reset)
	}
	fmt.Println("Code copied to clipboard.")
	return nil
}
