/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/configure
** File description:
** Configure flick source
 */

package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Flick-Corp/flick/internal/api/utils"
	"github.com/Flick-Corp/flick/internal/cli/config"
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

// getAnswer: Get the answer of the user.
//
// Returns:
// - result1 (string): The input.
func getAnswer() string {
	reader := bufio.NewReader(os.Stdin)
	changeServer, _ := reader.ReadString('\n')
	changeServer = strings.TrimSpace(changeServer)
	return changeServer
}

// needChanges: Check function to see if there is any changes.
//
// Returns:
// - result1 (bool): True or false.
func needChanges() bool {
	input := strings.ToLower(getAnswer())
	return input == "y" || input == "yes"
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
	var serverURL string
	var DefExpTime string
	var DefDownloadCount int32

	fmt.Printf("Change the default Flick server? [y/N]: ")
	if needChanges() {
		fmt.Printf("Enter the remote Flick server URL (e.g. https://flick.d3l.tech): ")
		fmt.Scan(&serverURL)
		config.Conf.ServerURL = config.NormalizeServerURL(serverURL) // TODO: verify input
	}

	fmt.Printf("Change the default expiration time? [y/N]: ")
	if needChanges() {
		fmt.Printf("Enter the default expiration time: ")
		fmt.Scan(&DefExpTime)
		config.Conf.DefExpTime = DefExpTime // TODO: verify input
	}

	fmt.Printf("Change the default download count? [y/N]: ")
	if needChanges() {
		fmt.Printf("Enter the default download count: ")
		fmt.Scan(&DefDownloadCount)
		if DefDownloadCount > 0 {
			config.Conf.DefDownloadCount = DefDownloadCount
		}
	}

	if err := config.Conf.SaveConfigurationFile(); err != nil {
		return err
	}

	fmt.Printf(utils.Green + "Configuration updated\n" + utils.Reset)
	return nil
}
