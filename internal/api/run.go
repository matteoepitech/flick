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
	"github.com/matteoepitech/flick/internal/api/identification"
	"github.com/matteoepitech/flick/internal/api/logging"
	"github.com/matteoepitech/flick/internal/api/path"
	"github.com/matteoepitech/flick/internal/api/routes"
	"github.com/matteoepitech/flick/internal/api/routes/account"
	"github.com/matteoepitech/flick/internal/api/routes/account/oauth"
	"github.com/matteoepitech/flick/internal/api/routes/files"
	"github.com/matteoepitech/flick/internal/api/routes/groups"
	groupsadmin "github.com/matteoepitech/flick/internal/api/routes/groups/admin"
	"github.com/matteoepitech/flick/internal/api/routes/users"
	usersadmin "github.com/matteoepitech/flick/internal/api/routes/users/admin"
)

// Constants
const addr string = ":15702"

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

// Run: Run the API over plain HTTP. TLS, HTTP/3 and same-origin routing are
// handled by the Caddy reverse proxy sitting in front of the API.
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

	pool, queries, err := setupDatabase(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/upload", files.UploadFileHandler(queries))
	mux.HandleFunc("/api/v1/identify", identification.IdentifyHandler(queries))
	mux.HandleFunc("/api/v1/download", files.DownloadFileHandler(queries))
	mux.HandleFunc("/api/v1/download/info", files.DownloadInfoHandler(queries))
	mux.HandleFunc("/api/v1/quota", files.QuotaHandler(queries))
	mux.HandleFunc("/api/v1/configure", routes.SendServerConfig())
	mux.HandleFunc("/api/v1/stats", routes.SendStats(queries))
	mux.HandleFunc("/api/v1/user-configure", routes.SendServerUserConfig())
	mux.HandleFunc("/api/v1/register", account.RegisterHandler(queries))
	mux.HandleFunc("/api/v1/login", account.LoginHandler(queries))
	mux.HandleFunc("/api/v1/whoami", account.WhoamiHandler(queries))
	mux.HandleFunc("/api/v1/device/code", oauth.DeviceCodeHandler(queries))
	mux.HandleFunc("/api/v1/device/token", oauth.DeviceTokenHandler(queries))
	mux.HandleFunc("/api/v1/device/approve", oauth.DeviceApproveHandler(queries))
	mux.HandleFunc("/api/v1/users/search", users.SearchUsersHandler(queries))
	mux.HandleFunc("/api/v1/admin/users", usersadmin.ListUsersHandler(queries))
	mux.HandleFunc("/api/v1/admin/users/{id}", usersadmin.UpdateUserHandler(queries))
	mux.HandleFunc("/api/v1/admin/groups", groupsadmin.GroupsHandler(queries))
	mux.HandleFunc("/api/v1/admin/groups/{id}", groupsadmin.GroupHandler(queries))
	mux.HandleFunc("/api/v1/admin/groups/{id}/members", groups.GroupMembersHandler(queries))
	mux.HandleFunc("/api/v1/admin/groups/{id}/members/{userId}", groups.GroupMemberHandler(queries))
	mux.HandleFunc("/api/v1/admin/groups/{id}/explore", groups.ExploreGroupHandler(queries))
	mux.HandleFunc("/api/v1/admin/groups/{id}/folders", groups.CreateGroupFolderHandler(queries))
	mux.HandleFunc("/api/v1/admin/groups/{id}/folders/{folderId}", groups.DeleteGroupFolderHandler(queries))
	routes.WriteDefaultConfig()

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Init the code cache from disk into RAM.
	if err := code.InitCodeCache(); err != nil {
		logging.LogInfoError("Cannot load code cache from disk: %v", err)
	}

	logging.LogInfoSuccess("FLICK server listening on %s", addr)

	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			logging.LogInfoError("HTTP server stopped: %v", err)
		}
	}()
	<-stopSignal

	return nil
}
