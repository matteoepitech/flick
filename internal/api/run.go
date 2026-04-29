/*
** FLICK PROJECT, 2026
** flick/internal/api/run
** File description:
** Flick API
 */

package api

import (
	"context"
	"fmt"
	"github.com/matteoepitech/flick/internal/api/utils"
	"github.com/quic-go/quic-go/http3"
	"io"
	"net/http"
	"os"
)

// Where the data is stored
var dataDir string

// uploadFileHandler: The upload file handler.
//
// Params:
// - w (http.ResponseWriter): The write channel.
// - r (*http.Request): The request headers informations.
func uploadFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "This endpoint is meant to be POST only", http.StatusNotFound)
		return
	}

	r.ParseMultipartForm(100 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		fmt.Printf(utils.Red + "[API]: Error while parsing an uploaded file\n" + utils.Reset)
		http.Error(w, "Cannot parse the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	dst, err := os.Create(dataDir + header.Filename)
	if err != nil {
		fmt.Printf(utils.Red+"[API]: Error while uploading a file of code <%s>\n"+utils.Reset, header.Filename)
		http.Error(w, "Cannot save the file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	fileBytes, err := io.Copy(dst, file)
	if err != nil {
		fmt.Printf(utils.Red+"[API]: Error while uploading a file of code <%s>\n"+utils.Reset, header.Filename)
		http.Error(w, "Error while copying the file", http.StatusInternalServerError)
		return
	}

	fmt.Printf(utils.Green+"[API]: Received a file with code <%s> (%d bytes)\n"+utils.Reset, header.Filename, fileBytes)
}

// downloadFileHandler: The download file handler.
//
// Params:
// - w (http.ResponseWriter): The write channel.
// - r (*http.Request): The request headers informations.
func downloadFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "This endpoint is meant to be GET only", http.StatusNotFound)
		return
	}

	query := r.URL.Query()
	code := query.Get("code")

	fmt.Printf("[API]: Trying to find the resource of the ID: <%s>\n", code)
	file, err := os.Stat(dataDir + code)
	if err != nil {
		fmt.Printf(utils.Red+"[API]: The resource <%s> is not found\n"+utils.Reset, code)
		http.Error(w, "Resource not found", http.StatusNotFound)
	}

	content, err := os.ReadFile(dataDir + code)
	if err != nil {
		fmt.Printf(utils.Red+"[API]: The resource <%s> can't be downloaded\n"+utils.Reset, code)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}

	w.Write(content)
	fmt.Printf(utils.Green+"[API]: File downloaded of %d bytes\n"+utils.Reset, file.Size())
}

// Run: Run the API on HTTP/3 (QUIC).
//
// Params:
// - ctx (context.Context): The context of the main.
//
// Returns:
// - result1 (error): nil if no error, otherwise an error.
func Run(ctx context.Context) error {
	http.HandleFunc("/upload", uploadFileHandler)
	http.HandleFunc("/download", downloadFileHandler)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf(utils.Red+"[API]: Unable to start the API: cannot get home dir: %w"+utils.Reset, err)
	}

	dataDir = homeDir + "/.flick/data/" // global var
	err = os.MkdirAll(dataDir, 0755)
	if err != nil {
		return fmt.Errorf(utils.Red+"[API]: Unable to start the API: cannot create the directory %s"+utils.Reset, dataDir)
	}

	fmt.Println("[API]: Starting FLICK server on :15702...")

	err = http3.ListenAndServeTLS(":15702", "certificates/cert.pem", "certificates/key.pem", nil)
	if err != nil {
		return fmt.Errorf(utils.Red+"[API]: Unable to start the API: server error: %w"+utils.Reset, err)
	}

	return nil
}
