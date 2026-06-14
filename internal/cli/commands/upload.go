/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/upload
** File description:
** Upload flick source
 */

package commands

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
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

	stat, err := os.Stat(args[0])
	if err != nil {
		return fmt.Errorf("Failure: Cannot get that file.")
	}

	serverLimits, err := config.GetServerLimits()
	if err != nil {
		return fmt.Errorf("Failure: Cannot get server limits: %w", err)
	}

	if serverLimits.MaxFileSizeMb > 0 && stat.Size() > int64(serverLimits.MaxFileSizeMb)*1024*1024 {
		return fmt.Errorf("Failure: The file is too large. The server only accepts files up to %d MB.", serverLimits.MaxFileSizeMb)
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

	fmt.Printf("Uploading the file %s... (%d bytes)\n", stat.Name(), stat.Size())

	file, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("Failure: Cannot open that file.")
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", stat.Name())
	if err != nil {
		return fmt.Errorf("Failure: Cannot create the form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return fmt.Errorf("Failure: Cannot copy that file.")
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
