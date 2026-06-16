/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/download
** File description:
** Download flick source
 */

package commands

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/matteoepitech/flick/internal/api/utils"
	"github.com/matteoepitech/flick/internal/cli/config"
	"github.com/matteoepitech/flick/internal/cli/network"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
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
	Items []downloadInfoItem `json:"items"`
}

// doDownloadRequest: Do the download request on the server.
//
// Params:
// - req (*http.Request): The request HTTP.
//
// Returns:
// - result1 (error): An error occured.
func doDownloadRequest(req *http.Request) error {
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

		if err := downloadArchive(part, bar); err != nil {
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
//
// Returns:
// - result1 (error): An error if occured.
func downloadArchive(part io.Reader, bar *progressbar.ProgressBar) error {
	tmp, err := os.CreateTemp("", "flick-download-*.zip")
	if err != nil {
		return fmt.Errorf("Failure: Cannot create temp archive: %w", err)
	}
	defer os.Remove(tmp.Name())

	proxyReader := io.TeeReader(part, bar)
	if _, err := io.Copy(tmp, proxyReader); err != nil {
		tmp.Close()
		return fmt.Errorf("Failure: Cannot download the archive: %w", err)
	}
	tmp.Close()

	return extractZip(tmp.Name(), ".")
}

// extractZip: Extract a zip archive into dest.
//
// Params:
// - zipPath (string): The path to the zip file on disk.
// - dest (string): The destination directory.
//
// Returns:
// - result1 (error): An error if occured.
func extractZip(zipPath string, dest string) error {
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return err
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("Failure: Cannot open the archive: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(absDest, f.Name)

		if target != absDest && !strings.HasPrefix(target, absDest+string(os.PathSeparator)) {
			return fmt.Errorf("Failure: unsafe path in archive: %q", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := writeZipEntry(f, target); err != nil {
			return err
		}
	}
	return nil
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
//
// Params:
// - code (string): The code to inspect.
//
// Returns:
// - result1 (downloadInfoResponse): The listing behind the code.
// - result2 (error): An error if occured.
func fetchDownloadInfo(code string) (downloadInfoResponse, error) {
	var info downloadInfoResponse

	req, err := http.NewRequest("GET", config.Conf.APIBaseURL()+"/download/info?code="+code, nil)
	if err != nil {
		return info, fmt.Errorf("Failure: Cannot create the request for the server.")
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
	if len(info.Items) == 0 {
		fmt.Println("This code holds no files.")
		return
	}
	fmt.Println("\nThis code contains:")
	for _, item := range info.Items {
		if item.IsFolder {
			fmt.Printf("  • "+utils.Blue+"%s/ (%d files, %s)\n"+utils.Reset, item.Name, item.FileCount, humanSize(item.Size))
		} else {
			fmt.Printf("  • "+utils.Dim+"%s (%s)\n"+utils.Reset, item.Name, humanSize(item.Size))
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
	code := strings.TrimSpace(codeLine)
	if code == "" {
		return fmt.Errorf("Failure: No code provided.")
	}
	fmt.Printf("Searching the code %s...\n", code)

	info, err := fetchDownloadInfo(code)
	if err != nil {
		return err
	}
	printDownloadInfo(info)

	fmt.Printf("%s [y/n]: ", "Download these files?")
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

	return doDownloadRequest(req)
}
