/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/login
** File description:
** Login flick source
 */

package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/Flick-Corp/flick/internal/cli/config"
	"github.com/Flick-Corp/flick/internal/cli/network"
	"github.com/Flick-Corp/flick/internal/utils/colors"
	"github.com/spf13/cobra"
)

// login CMD using cobra
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to the server",
	RunE:  RunLogin,
}

// deviceCodeResponse mirrors the server /device/code response body.
type deviceCodeResponse struct {
	DeviceCode string `json:"device_code"`
	UserCode   string `json:"user_code"`
	ExpiresIn  int    `json:"expires_in"`
	Interval   int    `json:"interval"`
}

// deviceTokenResponse mirrors the server /device/token response body.
type deviceTokenResponse struct {
	Status string `json:"status"`
	Token  string `json:"token"`
	UserID string `json:"user_id"`
}

// init: Init of the package goes here.
func init() {
	rootCmd.AddCommand(loginCmd)
}

// openBrowser: Best-effort open of a URL in the user's default browser. A
// failure is not fatal: the URL is always printed so the user can open it
// manually.
//
// Params:
// - url (string): The URL to open.
func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd, args = "cmd", []string{"/c", "start"}
	default:
		cmd = "xdg-open"
	}
	args = append(args, url)
	_ = exec.Command(cmd, args...).Start()
}

// requestDeviceCode: Ask the server for a fresh device authorization.
//
// Returns:
// - result1 (*deviceCodeResponse): The device code payload.
// - result2 (error): If something occured.
func requestDeviceCode() (*deviceCodeResponse, error) {
	resp, err := network.SharedClient.Post(config.Conf.APIBaseURL()+"/device/code", "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to reach the server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("server returned %s", resp.Status)
	}

	var result deviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode device code response: %w", err)
	}
	return &result, nil
}

// pollDeviceToken: Poll the server once for the session token of a device code.
// It returns the token response when approved, a nil response when still
// pending, or an error when the authorization was denied or expired.
//
// Params:
// - deviceCode (string): The opaque device code to poll with.
//
// Returns:
// - result1 (*deviceTokenResponse): The token when approved, nil when pending.
// - result2 (error): If denied, expired or unreachable.
func pollDeviceToken(deviceCode string) (*deviceTokenResponse, error) {
	body, err := json.Marshal(map[string]string{"device_code": deviceCode})
	if err != nil {
		return nil, err
	}

	resp, err := network.SharedClient.Post(config.Conf.APIBaseURL()+"/device/token", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to reach the server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %s", resp.Status)
	}

	var result deviceTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode device token response: %w", err)
	}

	if result.Token == "" {
		return nil, nil // still pending
	}
	return &result, nil
}

// RunLogin: Login to the server using the device authorization flow: open the
// activation page in the browser, then poll until the user approves it.
//
// Params:
// - cmd (*cobra.Command): The Cobra command.
// - args ([]string): Command arguments.
//
// Returns:
// - error: An error if occurred.
func RunLogin(cmd *cobra.Command, args []string) error {
	device, err := requestDeviceCode()
	if err != nil {
		return fmt.Errorf("failed to start login: %w", err)
	}

	activateURL := fmt.Sprintf("%s/activate?code=%s", config.NormalizeServerURL(config.Conf.ServerURL), device.UserCode)

	fmt.Printf("To log in, open the following page in your browser:\n\n")
	fmt.Printf("  %s%s%s\n\n", colors.Cyan, activateURL, colors.Reset)
	fmt.Printf("and confirm this code: %s%s%s\n\n", colors.Bold, device.UserCode, colors.Reset)
	openBrowser(activateURL)

	fmt.Printf("%sWaiting for approval...%s\n", colors.Dim, colors.Reset)

	deadline := time.Now().Add(time.Duration(device.ExpiresIn) * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(time.Duration(device.Interval) * time.Second)

		token, err := pollDeviceToken(device.DeviceCode)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		if token == nil {
			continue // still pending
		}

		if err := config.SaveCredentials(config.Credentials{UserID: token.UserID, Token: token.Token}); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		fmt.Printf(colors.Green + "You are now logged in!\n" + colors.Reset)
		return nil
	}

	return fmt.Errorf("login timed out, please run 'flick login' again")
}
