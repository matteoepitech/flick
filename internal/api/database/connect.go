/*
** FLICK PROJECT, 2026
** flick/internal/api/database/connect
** File description:
** PostgreSQL connection pool helper
 */

package database

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect: Open a PostgreSQL connection pool using the DATABASE_URL environment
// variable. The pool is safe for concurrent use and should be created once at
// startup and shared across the whole API.
//
// Params:
// - ctx (context.Context): The context of the caller.
//
// Returns:
// - *pgxpool.Pool: The ready-to-use connection pool.
// - error: nil on success, otherwise the connection error.
func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
