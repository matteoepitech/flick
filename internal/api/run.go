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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matteoepitech/flick/internal/api/code"
	"github.com/matteoepitech/flick/internal/api/database"
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

// setupDatabase: Setup the databse and return the pool and queries.
//
// Params:
// - ctx (context.Context): The context of the database.
//
// Returns:
// - result1 (*pgxpool.Pool): The pool of the PostgreSQL.
// - result2 (*database.Queries): The queries of the database.
// - result3 (error): An error if occured.
func setupDatabase(ctx context.Context) (*pgxpool.Pool, *database.Queries, error) {
	pool, err := database.Connect(ctx)
	if err != nil {
		return nil, nil, logging.LogInfoError("Cannot connect to database: %v", err)
	}
	logging.LogInfoSuccess("Connected to PostgreSQL!")

	return pool, database.New(pool), nil
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
		return logging.LogInfoError("Cannot create data directory %q: %v", path.GetDataDir(), err)
	}

	pool, _, err := setupDatabase(ctx) // TODO: THE _ IS NEEDED FOR HANDLE FUNC (FOR FUTURE ROUTES)
	if err != nil {
		return err
	}
	defer pool.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/upload", withCORS(routes.UploadFileHandler()))
	mux.HandleFunc("/download", withCORS(routes.DownloadFileHandler()))
	mux.HandleFunc("/configure", withCORS(routes.SendServerConfig()))
	mux.HandleFunc("/stats", withCORS(routes.SendStats()))
	mux.HandleFunc("/user-configure", withCORS(routes.SendServerUserConfig()))
	routes.WriteDefaultConfig()

	h3Server := &http3.Server{
		Addr:    addr,
		Handler: mux,
	}

	h2Handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := h3Server.SetQUICHeaders(w.Header()); err != nil {
			logging.LogInfoError("Cannot set QUIC headers: %v", err)
		}
		mux.ServeHTTP(w, r)
	})

	h2Server := &http.Server{
		Addr:    addr,
		Handler: h2Handler,
	}

	// Init the code cache from disk into RAM.
	if err := code.InitCodeCache(); err != nil {
		logging.LogInfoError("Cannot load code cache from disk: %v", err)
	}

	logging.LogInfoSuccess("FLICK server listening on %s", addr)

	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := h3Server.ListenAndServeTLS(certFile, keyFile); err != nil {
			logging.LogInfoError("HTTP/3 server stopped: %v", err)
		}
	}()
	go func() {
		if err := h2Server.ListenAndServeTLS(certFile, keyFile); err != nil {
			logging.LogInfoError("HTTP/2 server stopped: %v", err)
		}
	}()
	<-stopSignal

	return nil
}
