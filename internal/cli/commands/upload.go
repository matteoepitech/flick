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
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/tiagomelo/go-clipboard/clipboard"
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

	bodyString := string(body)
	fmt.Print("\nCode: " + bodyString + "\n")

	c := clipboard.New(clipboard.ClipboardOptions{Primary: true})
	if err := c.CopyText(bodyString); err != nil {
		return err
	}

	fmt.Println("Code copied to clipboard.")
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

	bar := progressbar.DefaultBytes(int64(body.Len()), "Uploading")
	progressBody := io.TeeReader(body, bar)

	req, err := http.NewRequest("POST", "https://"+serverIP+":15702/upload", progressBody)
	if err != nil {
		return fmt.Errorf("Failure: Cannot create the request for the server.\n")
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.ContentLength = int64(body.Len())

	return doUploadRequest(req)
}
