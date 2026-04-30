/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/download
** File description:
** Download flick source
 */

package commands

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

// createFile: Create the file of a multipartform.
//
// Params:
// - name (string): The name of the file.
// - data (string): The data of the file.
//
// Returns:
// - result1 (error): An error if something occured.
func createFile(name string, data string) error {
	fmt.Printf("Getting the file %s...\n", name)

	file, err := os.Create(name)
	if err != nil {
		return err
	}

	_, err = file.WriteString(data)
	if err != nil {
		return err
	}
	return nil
}

// doDownloadRequest: Do the download request on the server.
//
// Params:
// - req (*http.Request): The request HTTP.
//
// Returns:
// - result1 (error): An error occured.
func doDownloadRequest(req *http.Request) error {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Dev only: local self-signed cert.
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Failure: Cannot access the server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("Failure: Server returned %s", resp.Status)
	}

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
			data, err := io.ReadAll(part)
			if err != nil {
				return err
			}

			err = createFile(part.FileName(), string(data))
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

	req, err := http.NewRequest("GET", "https://"+serverIP+":15702/download?code="+code, body)
	if err != nil {
		return fmt.Errorf("Failure: Cannot create the request for the server.\n")
	}

	return doDownloadRequest(req)
}
