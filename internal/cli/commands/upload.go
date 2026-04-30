/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/upload
** File description:
** Upload flick source
 */

package commands

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

// doUploadRequest: Do the upload request on the server.
//
// Params:
// - req (*http.Request): The request HTTP.
//
// Returns:
// - result1 (error): An error occured.
func doUploadRequest(req *http.Request) error {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Dev only: local self-signed cert.
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Failure: Cannot access the server: %w\n", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("Failure: Server returned %s\n", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Failure: Invalid response from the server\n")
	}
	defer resp.Body.Close()

	fmt.Print("Code: " + string(body))
	return nil
}

// RunUpload: Run the upload command.
//
// Params:
// - cmd (*cobra.Command): The command.
//
// Returns:
// - result1 (error): An error if occured.
func RunUpload(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Failure: Internal CLI error.\n")
	}

	stat, err := os.Stat(args[0])
	if err != nil {
		return fmt.Errorf("Failure: Cannot get that file.\n")
	}
	fmt.Printf("Uploading the file %s... (%d bytes)\n", stat.Name(), stat.Size())

	file, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("Failure: Cannot open that file.\n")
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", stat.Name())
	if err != nil {
		return fmt.Errorf("Failure: Cannot create the form file: %w\n", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return fmt.Errorf("Failure: Cannot copy that file.\n")
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("Failure: Cannot finalize the upload body: %w\n", err)
	}

	req, err := http.NewRequest("POST", "https://"+serverIP+":15702/upload", body)
	if err != nil {
		return fmt.Errorf("Failure: Cannot create the request for the server.\n")
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	return doUploadRequest(req)
}
