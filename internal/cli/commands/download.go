/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/download
** File description:
** Download flick source
 */

package commands

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Flick-Corp/flick/internal/cli/config"
	"github.com/Flick-Corp/flick/internal/cli/network"
	archiveutil "github.com/Flick-Corp/flick/internal/utils/archive"
	"github.com/Flick-Corp/flick/internal/utils/checksum"
	"github.com/Flick-Corp/flick/internal/utils/colors"
	"github.com/Flick-Corp/flick/internal/utils/encryption"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// downloadInfoItem: one item behind a code.
type downloadInfoItem struct {
	Name      string `json:"name"`
	IsFolder  bool   `json:"isFolder"`
	FileCount int    `json:"fileCount"`
	Size      int64  `json:"size"`
}

// downloadInfoResponse: the listing returned by the /download/info endpoint.
type downloadInfoResponse struct {
	Items             []downloadInfoItem `json:"items"`
	Encrypted         bool               `json:"encrypted"`
	PasswordProtected bool               `json:"passwordProtected"`
	Message           string             `json:"message"`
}

// doDownloadRequest: Do the download request on the server.
//
// Params:
// - req (*http.Request): The request HTTP.
// - key (encryption.Key): The key used to decrypt the archive when decrypt is true.
// - decrypt (bool): Whether the downloaded archive must be decrypted before extraction.
//
// Returns:
// - result1 (error): An error occured.
func doDownloadRequest(req *http.Request, key encryption.Key, decrypt bool) error {
	resp, err := network.SharedClient.Do(req)
	if err != nil {
		return fmt.Errorf("Failure: Cannot access the server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Failure: %s", serverErrorMessage(body, resp.Status))
	}

	totalSizeStr := resp.Header.Get("X-Total-Size")
	totalSize, _ := strconv.ParseInt(totalSizeStr, 10, 64)
	if totalSize <= 0 {
		totalSize = -1
	}
	bar := progressbar.DefaultBytes(totalSize, "Downloading")

	contentType := resp.Header.Get("Content-Type")
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf("Failure: invalid Content-Type header: %w", err)
	}

	boundary, ok := params["boundary"]
	if !ok {
		return fmt.Errorf("Failure: missing multipart boundary in response")
	}

	reader := multipart.NewReader(resp.Body, boundary)

	expectedChecksum := resp.Header.Get("X-Flick-Checksum")

	// Uploads are always stored as a zip archive, so every "file" part is one
	// archive that we extract into the current directory.
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if part.FormName() != "file" {
			continue
		}

		if err := downloadArchive(part, bar, expectedChecksum, key, decrypt); err != nil {
			return err
		}
	}

	return nil
}

// downloadArchive: Buffer the archive to a temp file then extract into the current directory.
//
// Params:
// - part (io.Reader): The multipart file stream carrying the zip.
// - bar (*progressbar.ProgressBar): The progress bar to feed while downloading.
// - expectedChecksum (string): The archive's BLAKE3 digest announced by the server, or empty to skip integrity verification.
// - key (encryption.Key): The key used to decrypt the archive when decrypt is true.
// - decrypt (bool): Whether the archive is end-to-end encrypted and must be decrypted before extraction.
//
// Returns:
// - result1 (error): An error if occured.
func downloadArchive(part io.Reader, bar *progressbar.ProgressBar, expectedChecksum string, key encryption.Key, decrypt bool) error {
	tmp, err := os.CreateTemp("", "flick-download-*.zip")
	if err != nil {
		return fmt.Errorf("Failure: Cannot create temp archive: %w", err)
	}
	defer os.Remove(tmp.Name())

	hasher := checksum.New()
	proxyReader := io.TeeReader(part, io.MultiWriter(bar, hasher))
	if _, err := io.Copy(tmp, proxyReader); err != nil {
		tmp.Close()
		return fmt.Errorf("Failure: Cannot download the archive: %w", err)
	}
	tmp.Close()

	if expectedChecksum != "" {
		got := checksum.Sum(hasher)
		if !checksum.Equal(got, expectedChecksum) {
			return fmt.Errorf("Failure: checksum mismatch, the downloaded file is corrupted (expected %s, got %s)", expectedChecksum, got)
		}
	}

	zipPath := tmp.Name()
	if decrypt {
		decPath, err := decryptToTemp(tmp.Name(), key)
		if err != nil {
			return err
		}
		defer os.Remove(decPath)
		zipPath = decPath
	}

	_, err = archiveutil.Extract(zipPath, ".")
	return err
}

// decryptToTemp: Decrypt the archive at srcPath into a new temporary file under
// key, returning that file's path. The caller is responsible for removing it.
//
// Params:
// - srcPath (string): The ciphertext archive to decrypt.
// - key (encryption.Key): The key recovered from the share code.
//
// Returns:
// - result1 (string): The path to the temporary plaintext archive.
// - result2 (error): An error if occured.
func decryptToTemp(srcPath string, key encryption.Key) (string, error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("Failure: Cannot open the download: %w", err)
	}
	defer src.Close()

	tmp, err := os.CreateTemp("", "flick-decrypt-*.zip")
	if err != nil {
		return "", fmt.Errorf("Failure: Cannot create temp archive: %w", err)
	}

	if err := encryption.Decrypt(tmp, src, key); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", fmt.Errorf("Failure: Cannot decrypt the archive: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("Failure: Cannot finalize the decrypted archive: %w", err)
	}
	return tmp.Name(), nil
}

// humanSize: Format a byte count into a short human-readable string.
//
// Params:
// - bytes (int64): The size in bytes.
//
// Returns:
// - result1 (string): The formatted size (e.g. "1.5 MB").
func humanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// fetchDownloadInfo: List what is behind a code without consuming the download.
// When the code is password protected, the listing is only revealed if password
// is the correct one; an empty or wrong password yields a placeholder listing
// with PasswordProtected set.
//
// Params:
// - code (string): The code to inspect.
// - password (string): The download password, or empty when none is known yet.
//
// Returns:
// - result1 (downloadInfoResponse): The listing behind the code.
// - result2 (error): An error if occured.
func fetchDownloadInfo(code string, password string) (downloadInfoResponse, error) {
	var info downloadInfoResponse

	req, err := http.NewRequest("GET", config.Conf.APIBaseURL()+"/download/info?code="+code, nil)
	if err != nil {
		return info, fmt.Errorf("Failure: Cannot create the request for the server.")
	}
	if password != "" {
		req.Header.Set("X-Flick-Password", password)
	}

	resp, err := network.SharedClient.Do(req)
	if err != nil {
		return info, fmt.Errorf("Failure: Cannot access the server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		return info, fmt.Errorf("Failure: %s", serverErrorMessage(body, resp.Status))
	}

	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return info, fmt.Errorf("Failure: Invalid response from the server")
	}
	return info, nil
}

// printDownloadInfo: Show the files behind a code before the consuming download.
//
// Params:
// - info (downloadInfoResponse): The listing to print.
func printDownloadInfo(info downloadInfoResponse) {
	if info.Message != "" {
		fmt.Printf("\n"+colors.Yellow+"Message from the sender:"+colors.Reset+" %s\n", info.Message)
	}
	if len(info.Items) == 0 {
		fmt.Println("This code holds no files.")
		return
	}
	fmt.Println("\nThis code contains:")
	for _, item := range info.Items {
		if item.IsFolder {
			fmt.Printf("  • "+colors.Blue+"%s/ (%d files, %s)\n"+colors.Reset, item.Name, item.FileCount, humanSize(item.Size))
		} else {
			fmt.Printf("  • "+colors.Dim+"%s (%s)\n"+colors.Reset, item.Name, humanSize(item.Size))
		}
	}
}

// RunDownload: Run the download command.
//
// Params:
// - cmd (*cobra.Command): The command.
//
// Returns:
// - result1 (error): An error if occured.
func RunDownload(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Specify the code: ")
	codeLine, _ := reader.ReadString('\n')
	input := strings.TrimSpace(codeLine)
	if input == "" {
		return fmt.Errorf("Failure: No code provided.")
	}

	// An encrypted Flick carries its decryption key after a "#", e.g.
	// "ocean-tiger-42#<key>". Only the code is ever sent to the server.
	code, key, hasKey, err := splitCode(input)
	if err != nil {
		return err
	}
	fmt.Printf("Searching the code %s...\n", code)

	info, err := fetchDownloadInfo(code, "")
	if err != nil {
		return err
	}

	var password string
	if info.PasswordProtected {
		password, err = promptPassword()
		if err != nil {
			return err
		}
		info, err = fetchDownloadInfo(code, password)
		if err != nil {
			return err
		}
		if info.PasswordProtected {
			return fmt.Errorf("Failure: Wrong password for this code.")
		}
	}

	if info.Encrypted && !hasKey {
		return fmt.Errorf("Failure: This Flick is end-to-end encrypted. Use the full code including the part after #.")
	}
	if info.Encrypted {
		fmt.Println(colors.Dim + "This content is end-to-end encrypted; it will be decrypted locally." + colors.Reset)
	}
	printDownloadInfo(info)

	fmt.Printf("%s [y/N]: ", "Download these files?")
	line, _ := reader.ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer != "y" && answer != "yes" {
		fmt.Println("Aborted.")
		return nil
	}

	req, err := http.NewRequest("GET", config.Conf.APIBaseURL()+"/download?code="+code, &bytes.Buffer{})
	if err != nil {
		return fmt.Errorf("Failure: Cannot create the request for the server.")
	}
	if password != "" {
		req.Header.Set("X-Flick-Password", password)
	}

	return doDownloadRequest(req, key, info.Encrypted)
}

// promptPassword: Read a download password from the terminal without echoing it.
// Falls back to a plain (visible) read when stdin is not a terminal, e.g. piped.
//
// Returns:
// - result1 (string): The entered password.
// - result2 (error): An error if reading failed.
func promptPassword() (string, error) {
	fmt.Printf("This code is password protected. Enter the password: ")

	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		raw, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("Failure: Cannot read the password: %w", err)
		}
		return strings.TrimSpace(string(raw)), nil
	}

	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line), nil
}

// splitCode: Separate a share code from its optional decryption key. A code of
// the form "code#key" yields the bare code plus the decoded key; a plain code
// yields the code with no key.
//
// Params:
// - input (string): The code typed by the user, possibly with a "#key" suffix.
//
// Returns:
// - result1 (string): The bare share code to send to the server.
// - result2 (encryption.Key): The decoded key, valid only when result3 is true.
// - result3 (bool): True when a key was present in the input.
// - result4 (error): An error if the key part is malformed.
func splitCode(input string) (string, encryption.Key, bool, error) {
	i := strings.IndexByte(input, '#')
	if i == -1 {
		return input, encryption.Key{}, false, nil
	}

	key, err := encryption.DecodeKey(input[i+1:])
	if err != nil {
		return "", encryption.Key{}, false, fmt.Errorf("Failure: Invalid decryption key in the code: %w", err)
	}
	return input[:i], key, true, nil
}
