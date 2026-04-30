/*
** FLICK PROJECT, 2026
** flick/cmd/cli/main
** File description:
** CLI Main file
 */

package main

import (
	"context"
	"github.com/matteoepitech/flick/internal/cli"
	"log"
)

// main Main entry point
func main() {
	err := cli.Run(context.Background())

	if err != nil {
		log.Fatal(err)
	}
}
