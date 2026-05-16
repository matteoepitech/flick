/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/configure
** File description:
** Configure flick source
 */

package commands

import (
	"fmt"

	"github.com/matteoepitech/flick/internal/api/utils"
	"github.com/matteoepitech/flick/internal/cli/config"
	"github.com/spf13/cobra"
)

// Configure CMD using cobra
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure the Flick CLI",
	RunE:  RunConfigure,
}

// init: Init of the package goes here.
func init() {
	rootCmd.AddCommand(configureCmd)
}

// needChanges: Check function to see if there is any changes.
//
// Params:
// - input (string): The input of the changes.
//
// Returns:
// - result1 (bool): True or false.
func needChanges(input string) bool {
	return input == "y" || input == "Y"
}

// RunConfigure: Configure the CLI default settings.
//
// Params:
// - cmd (*cobra.Command): The Cobra command.
// - args ([]string): Command arguments.
//
// Returns:
// - error: An error if occurred.
func RunConfigure(cmd *cobra.Command, args []string) error {
	var changeServer string
	var changeDefExpTime string
	var changeMaxExpTime string
	var serverIP string
	var DefExpTime string
	var MaxExpTime string

	fmt.Printf("Change the default Flick server? (y/n): ")
	fmt.Scan(&changeServer)
	if needChanges(changeServer) {
		fmt.Printf("Enter the remote Flick server (IP/DNS): ")
		fmt.Scan(&serverIP)
		config.Conf.ServerIP = serverIP // TODO: verify input
	}

	fmt.Printf("Change the default expiration time? (y/n): ")
	fmt.Scan(&changeDefExpTime)
	if needChanges(changeDefExpTime) {
		fmt.Printf("Enter the default expiration time: ")
		fmt.Scan(&DefExpTime)
		config.Conf.DefExpTime = DefExpTime // TODO: verify input
	}

	fmt.Printf("Change the maximum expiration time? (y/n): ")
	fmt.Scan(&changeMaxExpTime)
	if needChanges(changeMaxExpTime) {
		fmt.Printf("Enter the maximum expiration time: ")
		fmt.Scan(&MaxExpTime)
		config.Conf.MaxExpTime = MaxExpTime // TODO: verify input
	}

	if err := config.Conf.SaveConfigurationFile(); err != nil {
		return err
	}

	fmt.Printf(utils.Green + "Configuration updated\n" + utils.Reset)
	return nil
}
