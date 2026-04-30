/*
** FLICK PROJECT, 2026
** flick/internal/cli/run
** File description:
** CLI run package
 */

package cli

import (
	"context"
	"github.com/matteoepitech/flick/internal/cli/commands"
)

// Run: Run the CLI.
//
// Params:
// - ctx (context.Context): The context of the main.
//
// Returns:
// - result1 (error): nil if no error, otherwise an error.
func Run(ctx context.Context) error {
	return commands.Execute(ctx)
}
