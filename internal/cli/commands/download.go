/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/download
** File description:
** Download flick source
 */

package commands

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	"github.com/matteoepitech/flick/internal/cli/config"
	"github.com/matteoepitech/flick/internal/cli/network"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

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

	// Looking for the multipart form "file"
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if part.FormName() == "file" {
			file, err := os.Create(part.FileName())
			if err != nil {
				return err
			}

			proxyReader := io.TeeReader(part, bar)
			_, err = io.Copy(file, proxyReader)
			file.Close()

			if err != nil {
				return err
			}
		}
	}

	return nil
}

// RunDownload: Run the download command.
//
// Params:
// - cmd (*cobra.Command): The command.
//
// Returns:
// - result1 (error): An error if occured.
func RunDownload(cmd *cobra.Command, args []string) error {
	var code string

	fmt.Printf("Specify the code: ")
	fmt.Scan(&code)
	fmt.Printf("Searching the code %s...\n", code)

	body := &bytes.Buffer{}

	req, err := http.NewRequest("GET", config.Conf.APIBaseURL()+"/download?code="+code, body)
	if err != nil {
		return fmt.Errorf("Failure: Cannot create the request for the server.")
	}

	return doDownloadRequest(req)
}
