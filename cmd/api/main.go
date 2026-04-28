/*
** FLICK PROJECT, 2026
** flick/cmd/api/main
** File description:
** MAIN entry point for the API binary
 */

package main

import (
	"context"
	"github.com/matteoepitech/flick/internal/api"
	"log"
)

// main Main entry point
func main() {
	err := api.Run(context.Background())

	if err != nil {
		log.Fatal(err)
	}
}
