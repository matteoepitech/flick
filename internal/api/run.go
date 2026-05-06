/*
** FLICK PROJECT, 2026
** flick/internal/api/run
** File description:
** Flick API
 */

package api

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/matteoepitech/flick/internal/api/logging"
	"github.com/matteoepitech/flick/internal/api/metadata"
	"github.com/matteoepitech/flick/internal/api/routes"
	"github.com/quic-go/quic-go/http3"
)

// Where the data is stored
var dataDir string

// Logging for the API
var logger logging.Logger = logging.Logger{
	Prefix: "API",
}

// Run: Run the API on HTTP/3 (QUIC).
//
// Params:
// - ctx (context.Context): The context of the main.
//
// Returns:
// - result1 (error): nil if no error, otherwise an error.
func Run(ctx context.Context) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return logger.InfoError("Unable to start the API, cannot get the HOME directory")
	}

	dataDir = homeDir + "/.flick/data/" // global var
	err = os.MkdirAll(dataDir, 0755)
	if err != nil {
		return logger.InfoError("Unable to start the API, cannot create the directory %s", dataDir)
	}

	http.HandleFunc("/upload", routes.UploadFileHandler(dataDir, logger))
	http.HandleFunc("/download", routes.DownloadFileHandler(dataDir, logger))
	logger.InfoSuccess("Starting FLICK server on port 15702...")

	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, os.Interrupt, syscall.SIGTERM)
	go metadata.CheckExpiration(dataDir, logger)
	<-stopSignal

	err = http3.ListenAndServeTLS(":15702", "certificates/cert.pem", "certificates/key.pem", nil)
	if err != nil {
		return logger.InfoError("Unable to start the API with server error: %s", err.Error())
	}

	return nil
}
