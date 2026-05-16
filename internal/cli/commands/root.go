/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/root
** File description:
** Commands root file
 */

package commands

import (
	"context"
	"github.com/spf13/cobra"
)

// Root CMD using cobra
var rootCmd = &cobra.Command{
	Use:          "flick-cli",
	Args:         cobra.ArbitraryArgs,
	RunE:         runCLI,
	SilenceUsage: true,
}

// The default server IP.
var serverIP string = "127.0.0.1"

func init() {
	rootCmd.Flags().String("exp", "1d", "Expiration duration")
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

	for _, sub := range cmd.Commands() {
		if sub.Name() == args[0] {
			return cmd.Help()
		}

	}
	return RunUpload(cmd, args, exp)
}
