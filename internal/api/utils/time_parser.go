/*
** FLICK PROJECT, 2026
** flick/internal/api/utils/time_parser
** File description:
** Timer parser source file
 */

package utils

import (
	"fmt"
	"strconv"
	"time"
)

// ParseExpirationTime: Parse the given expiration time.
//
// Params:
// - exp (string): The expiration string to parse.
//
// Returns:
// - result1 (time.Time): The time result.
// - result2 (error): An error if something occured.
func ParseExpirationTime(exp string) (time.Time, error) {
	if duration, err := time.ParseDuration(exp); err == nil {
		return time.Now().Add(duration), nil
	}

	if len(exp) < 2 {
		return time.Time{}, fmt.Errorf("expiration too short: %s", exp)
	}

	unit := exp[len(exp)-1]
	value, err := strconv.Atoi(exp[:len(exp)-1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid number in expiration %s: %w", exp, err)
	}

	switch unit {
	case 'd':
		return time.Now().AddDate(0, 0, value), nil
	case 'w':
		return time.Now().AddDate(0, 0, value*7), nil
	case 'M':
		return time.Now().AddDate(0, value, 0), nil
	case 'y':
		return time.Now().AddDate(value, 0, 0), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported time unit %c", unit)
	}
}
