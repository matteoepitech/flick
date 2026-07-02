/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/login
** File description:
** Login flick source
 */

package commands

import (
	"fmt"

	"github.com/Flick-Corp/flick/internal/cli/config"
	"github.com/Flick-Corp/flick/internal/utils/colors"
	"github.com/spf13/cobra"
)

// logout CMD using cobra
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout your account",
	RunE:  RunLogout,
}

// init: Init of the package goes here.
func init() {
	rootCmd.AddCommand(logoutCmd)
}

// RunLogout: Logout.
//
// Params:
// - cmd (*cobra.Command): The Cobra command.
// - args ([]string): Command arguments.
//
// Returns:
// - error: An error if occurred.
func RunLogout(cmd *cobra.Command, args []string) error {
	creds, err := config.LoadCredentials()
	if err != nil || creds == nil || creds.Token == "" || creds.UserID == "" {
		fmt.Printf("You are not logged in.\n")
		return nil
	}

	creds.Token = ""
	if err := config.SaveCredentials(*creds); err != nil {
		return fmt.Errorf("you are not able to be logged out.")
	}

	fmt.Printf(colors.Dim + "You are now logged out.\n" + colors.Reset)
	return nil

}
