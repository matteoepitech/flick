/*
** FLICK PROJECT, 2026
** flick/internal/api/logging/logging
** File description:
** Logging go file
 */

package logging

import (
	"fmt"
	"github.com/matteoepitech/flick/internal/api/utils"
	"time"
)

// Logger structure
type Logger struct {
	Prefix string
}

// InfoSuccess: Print a success log message.
//
// Params:
// - format (string): The format string (printf style).
// - args (...any): The arguments for formatting.
func (l Logger) InfoSuccess(format string, args ...any) {
	now := time.Now().Format("15:04:05")
	msg := fmt.Sprintf(format, args...)

	fmt.Printf("[%s] "+utils.Green+"[%s] %s\n"+utils.Reset, now, l.Prefix, msg)
}

// InfoError: Print an error log message.
//
// Params:
// - format (string): The format string (printf style).
// - args (...any): The arguments for formatting.
//
// Returns:
// - error: The formatted error.
func (l Logger) InfoError(format string, args ...any) error {
	now := time.Now().Format("15:04:05")
	msg := fmt.Sprintf(format, args...)

	fmt.Printf("[%s] "+utils.Red+"[%s] %s\n"+utils.Reset, now, l.Prefix, msg)
	return fmt.Errorf("%s: %s", l.Prefix, msg)
}

// Info: Print a log message.
//
// Params:
// - format (string): The format string (printf style).
// - args (...any): The arguments for formatting.
func (l Logger) Info(format string, args ...any) {
	now := time.Now().Format("15:04:05")
	msg := fmt.Sprintf(format, args...)

	fmt.Printf("[%s] [%s] %s\n", now, l.Prefix, msg)
}
