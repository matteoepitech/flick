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

	"github.com/matteoepitech/flick/internal/api/code"
	"github.com/matteoepitech/flick/internal/api/logging"
	"github.com/matteoepitech/flick/internal/api/path"
	"github.com/matteoepitech/flick/internal/api/routes"
	"github.com/quic-go/quic-go/http3"
)

// Constants
const addr string = ":15702"
const certFile string = "certificates/cert.pem"
const keyFile string = "certificates/key.pem"

// withCORS: Wrap a handler with permissive CORS headers so browsers on any origin can call the
// API. Handles the preflight OPTIONS request transparently.
//
// Params:
// - next (http.HandlerFunc): The wrapped handler.
//
// Returns:
// - http.HandlerFunc: The wrapping handler with CORS headers applied.
func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Expose-Headers", "X-Total-Size, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

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
	if err := os.MkdirAll(path.GetDataDir(), 0755); err != nil {
		return logger.InfoError("Unable to start the API, cannot create the directory %s", path.GetDataDir())
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/upload", withCORS(routes.UploadFileHandler(logger)))
	mux.HandleFunc("/download", withCORS(routes.DownloadFileHandler(logger)))
	mux.HandleFunc("/configure", withCORS(routes.SendServerConfig(logger)))
	mux.HandleFunc("/user-configure", withCORS(routes.SendServerUserConfig(logger)))
	routes.WriteDefaultConfig(logger)

	h3Server := &http3.Server{
		Addr:    addr,
		Handler: mux,
	}

	h2Handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := h3Server.SetQUICHeaders(w.Header()); err != nil {
			logger.InfoError("Cannot set QUIC headers: %s", err.Error())
		}
		mux.ServeHTTP(w, r)
	})

	h2Server := &http.Server{
		Addr:    addr,
		Handler: h2Handler,
	}

	// Init the code cache from disk into RAM.
	code.InitCodeCache()

	logger.InfoSuccess("Starting FLICK server on port 15702...")

	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := h3Server.ListenAndServeTLS(certFile, keyFile); err != nil {
			logger.InfoError("HTTP/3 server stopped: %s", err.Error())
		}
	}()
	go func() {
		if err := h2Server.ListenAndServeTLS(certFile, keyFile); err != nil {
			logger.InfoError("HTTP/2 server stopped: %s", err.Error())
		}
	}()
	<-stopSignal

	return nil
}
