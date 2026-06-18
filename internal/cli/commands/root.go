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

	"github.com/matteoepitech/flick/internal/cli/config"
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
	rootCmd.Flags().String("exp", config.Conf.DefExpTime, "Expiration duration")
	rootCmd.Flags().String("mdc", strconv.FormatInt(int64(config.Conf.DefDownloadCount), 10), "Max download count")
	rootCmd.Flags().BoolP("encrypt", "e", false, "Encrypt the upload end-to-end")
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
	mdc, _ := cmd.Flags().GetString("mdc")
	encrypt, _ := cmd.Flags().GetBool("encrypt")

	for _, sub := range cmd.Commands() {
		if sub.Name() == args[0] {
			return cmd.Help()
		}

	}
	return RunUpload(cmd, args, exp, mdc, encrypt)
}
