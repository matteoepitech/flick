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
		http.Error(w, "Cannot parse the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	dst, err := os.Create(dataDir + "/" + header.Filename)
	if err != nil {
		http.Error(w, "Cannot save the file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	fileBytes, err := io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Error while copying the file", http.StatusInternalServerError)
		return
	}

	fmt.Printf("[API]: Received a file of %d bytes\n", fileBytes)
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

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf(utils.Red+"cannot get home dir: %w"+utils.Reset, err)
	}

	dataDir = homeDir + "/.flick/data" // global var
	err = os.MkdirAll(dataDir, 0755)
	if err != nil {
		return fmt.Errorf(utils.Red + "[API]: cannot create the directory /opt/flick/data/" + utils.Reset)
	}

	fmt.Println("[API]: Starting FLICK server on :15702...")

	err = http3.ListenAndServeTLS(":15702", "certificates/cert.pem", "certificates/key.pem", nil)
	if err != nil {
		return fmt.Errorf(utils.Red+"[API]: server error: %w"+utils.Reset, err)
	}

	return nil
}
