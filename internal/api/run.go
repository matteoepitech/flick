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
	"time"
)

// Run: Run the API.
//
// Params:
// - ctx (context.Context): The context of the main
//
// Returns:
// - result1 (error): nil if no error, otherwise an error.
func Run(ctx context.Context) error {
	fmt.Println("API: start")

	select {
	case <-time.After(2 * time.Second):
		fmt.Println("API: done")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("api canceled: %w", ctx.Err())
	}
}
