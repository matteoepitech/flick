/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/root
** File description:
** Commands root file
 */

package commands

import (
	"context"
	"strconv"

	"github.com/Flick-Corp/flick/internal/cli/config"
	"github.com/spf13/cobra"
)

// Root CMD using cobra
var rootCmd = &cobra.Command{
	Use:          "flick",
	Args:         cobra.ArbitraryArgs,
	RunE:         runCLI,
	SilenceUsage: true,
}

// init: Init root.
func init() {
	rootCmd.Flags().StringP("exp", "x", config.Conf.DefExpTime, "Expiration duration")
	rootCmd.Flags().StringP("mdc", "d", strconv.FormatInt(int64(config.Conf.DefDownloadCount), 10), "Max download count")
	rootCmd.Flags().BoolP("encrypt", "e", false, "Encrypt the upload end-to-end")
	rootCmd.Flags().StringP("password", "p", "", "Protect the download with a password")
	rootCmd.Flags().StringP("message", "m", "", "Attach a personal message shown to the downloader")
}

// Execute: Execute the root command.
//
// Params:
// - ctx (context.Context): The context of the program.
//
// Returns:
// - result1 (error): An error if something occured.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

// runCLI: Run the CLI command.
//
// Params:
// - cmd (*cobra.Command): The actual command done by the user.
// - args ([]string): The args.
//
// Returns:
// - result1 (error): An error if something occured.
func runCLI(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return RunDownload(cmd, args)
	}

	exp, _ := cmd.Flags().GetString("exp")
	if !cmd.Flags().Changed("exp") {
		exp = ""
	}
	mdc, _ := cmd.Flags().GetString("mdc")
	if !cmd.Flags().Changed("mdc") {
		mdc = ""
	}
	encrypt, _ := cmd.Flags().GetBool("encrypt")
	password, _ := cmd.Flags().GetString("password")
	message, _ := cmd.Flags().GetString("message")

	for _, sub := range cmd.Commands() {
		if sub.Name() == args[0] {
			return cmd.Help()
		}

	}
	return RunUpload(cmd, args, exp, mdc, encrypt, password, message)
}
