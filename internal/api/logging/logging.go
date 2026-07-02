/*
** FLICK PROJECT, 2026
** flick/internal/api/logging/logging
** File description:
** Logging go file
 */

package logging

import (
	"fmt"
	"github.com/Flick-Corp/flick/internal/utils/colors"
	"time"
)

// printLogLabel: Print a label in this format <date> [<title>] <subtitle> > .
//
// Params:
// - title (string): The title.
// - titleColor (string): The title color.
// - subtitle (string): The subtitle title.
// - subtitleColor (string): The subtitle title color.
func printLogLabel(title string, titleColor string, subtitle string, subtitleColor string) {
	now := time.Now().Format("15:04:05")

	fmt.Printf(colors.Dim+"%s"+colors.Reset+" "+
		colors.Gray+"["+colors.Reset+titleColor+colors.Bold+"%s"+colors.Reset+colors.Gray+"]"+colors.Reset+" "+
		subtitleColor+colors.Bold+"%s"+colors.Reset+colors.Gray+" > "+colors.Reset,
		now, title, subtitle)
}

// LogInfoSuccess: Print a success log message.
//
// Params:
// - format (string): The format string (printf style).
// - args (...any): The arguments for formatting.
func LogInfoSuccess(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)

	printLogLabel("SUCCESS", colors.BrightGreen, "INFO", colors.BrightWhite)
	fmt.Printf("%s\n", msg)
}

// LogInfoError: Print an error log message.
//
// Params:
// - format (string): The format string (printf style).
// - args (...any): The arguments for formatting.
//
// Returns:
// - error: The formatted error.
func LogInfoError(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)

	printLogLabel("ERROR", colors.BrightRed, "INFO", colors.BrightWhite)
	fmt.Printf("%s\n", msg)
	return fmt.Errorf("%s", msg)
}

// LogInfo: Print a log message.
//
// Params:
// - format (string): The format string (printf style).
// - args (...any): The arguments for formatting.
func LogInfo(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)

	printLogLabel("INFO", colors.BrightBlue, "INFO", colors.BrightWhite)
	fmt.Printf("%s\n", msg)
}
