/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/whoami
** File description:
** Whoami flick source
 */

package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/matteoepitech/flick/internal/cli/config"
	"github.com/matteoepitech/flick/internal/cli/network"
	"github.com/spf13/cobra"
)

// Configure CMD using cobra
var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show who you are if logged in",
	RunE:  RunWhoami,
}

// whoamiRequest mirrors the server /whoami request body.
type whoamiRequest struct {
	Token string `json:"token"`
}

// whoamiUser mirrors the user object returned by /whoami.
type whoamiUser struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

// whoamiResponse mirrors the server /whoami response body.
type whoamiResponse struct {
	User whoamiUser `json:"user"`
}

// init: Init of the package goes here.
func init() {
	rootCmd.AddCommand(whoamiCmd)
}

// RunWhoami: Whoami the CLI default settings.
//
// Params:
// - cmd (*cobra.Command): The Cobra command.
// - args ([]string): Command arguments.
//
// Returns:
// - error: An error if occurred.
func RunWhoami(cmd *cobra.Command, args []string) error {
	creds, err := config.EnsureCredentials()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}
	if creds.Token == "" {
		return fmt.Errorf("you are not logged in")
	}

	body, err := json.Marshal(whoamiRequest{Token: creds.Token})
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, config.Conf.APIBaseURL()+"/whoami", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := network.SharedClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach the server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s", resp.Status)
	}

	var result whoamiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Printf("Username: %s\nEmail:    %s\nID:       %s\n", result.User.Username, result.User.Email, result.User.ID)
	return nil
}
