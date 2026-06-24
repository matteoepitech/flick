/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/explore/explore
** File description:
** Interactive group explorer command (flick explore): wires the Bubble Tea
** program that browses groups, navigates folders, downloads and manages files.
 */

package explore

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matteoepitech/flick/internal/cli/config"
	"github.com/spf13/cobra"
)

// exploreCmd: the `flick explore` subcommand.
var exploreCmd = &cobra.Command{
	Use:   "explore",
	Short: "Browse your groups and their files interactively",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunExplore()
	},
}

// init: Register the explore subcommand.
func init() {
	rootCmd.AddCommand(exploreCmd)
}

// RunExplore: Launch the interactive group explorer.
//
// Returns:
// - result1 (error): An error if credentials are missing or the program failed.
func RunExplore() error {
	creds, err := config.EnsureCredentials()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}
	if creds.Token == "" {
		return fmt.Errorf("you are not logged in")
	}

	model := exploreModel{
		token:  creds.Token,
		mode:   modeGroups,
		status: "Loading...",
	}
	if _, err := tea.NewProgram(model).Run(); err != nil {
		return fmt.Errorf("failed to run explorer: %w", err)
	}
	return nil
}
