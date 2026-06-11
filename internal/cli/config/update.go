/*
** FLICK PROJECT, 2026
** flick/internal/cli/config/update
** File description:
** Update source file
 */

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/schollz/progressbar/v3"
)

// Release response used with the request.
type ReleaseResponse struct {
	Version         string `json:"version"`
	Commit          string `json:"commit"`
	BuildDate       string `json:"build_date"`
	URLLinuxAMD64   string `json:"url_linux_amd64"`
	URLLinuxARM64   string `json:"url_linux_arm64"`
	URLDarwinAMD64  string `json:"url_darwin_amd64"`
	URLDarwinARM64  string `json:"url_darwin_arm64"`
	URLWindowsAMD64 string `json:"url_windows_amd64"`
}

// needUpdates: Check function to ask user to do update or not.
//
// Params:
// - input (string): The input of the updates.
//
// Returns:
// - result1 (bool): True or false.
func needUpdates(input string) bool {
	return input == "y" || input == "Y"
}

// getReleaseVersion: Get the version of the release.
//
// Params:
// - release (ReleaseResponse): The release structure.
//
// Returns:
// - result1 (string): The string url.
func getReleaseVersion(release ReleaseResponse) string {
	var downloadURL string

	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "darwin/arm64":
		downloadURL = release.URLDarwinARM64
	case "darwin/amd64":
		downloadURL = release.URLDarwinAMD64
	case "linux/amd64":
		downloadURL = release.URLLinuxAMD64
	case "linux/arm64":
		downloadURL = release.URLLinuxARM64
	case "windows/amd64":
		downloadURL = release.URLWindowsAMD64
	}
	return downloadURL
}

// UpdateNewVersion: Update the CLI to the new version.
//
// Params:
// - release (ReleaseResponse): The release version.
//
// - result1 (error): Error or nil.
func UpdateNewVersion(release ReleaseResponse) error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	resp, err := http.Get(getReleaseVersion(release))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cannot download the new version: server returned %s", resp.Status)
	}

	tmpFile, err := os.Create(execPath + ".tmp")
	if err != nil {
		return err
	}

	bar := progressbar.DefaultBytes(resp.ContentLength, "Updating")
	_, err = io.Copy(io.MultiWriter(tmpFile, bar), resp.Body)
	if err != nil {
		tmpFile.Close()
		return err
	}

	err = tmpFile.Close()
	if err != nil {
		return err
	}

	err = os.Chmod(execPath+".tmp", 0755)
	if err != nil {
		return err
	}

	err = os.Rename(execPath+".tmp", execPath)
	if err != nil {
		return err
	}

	return nil
}

// CheckUpdate: Check any update in apt.d3l.tech.
//
// Params:
// - currentVersion (string): The current CLI version.
func CheckUpdate(currentVersion string) {
	body := &bytes.Buffer{}
	req, err := http.NewRequest("GET", "https://apt.d3l.tech/releases/version.json", body)
	if err != nil {
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var release ReleaseResponse
	err = json.NewDecoder(resp.Body).Decode(&release)
	if err != nil {
		return
	}

	if currentVersion != release.Version {
		var autoUpdateQuestion string

		fmt.Println("🎉 New update available! (" + currentVersion + " -> " + release.Version + ")")
		fmt.Printf("Do you want to auto-update? (y/n): ")
		fmt.Scan(&autoUpdateQuestion)
		if needUpdates(autoUpdateQuestion) {
			if UpdateNewVersion(release) != nil {
				fmt.Println("Failure: Cannot update to the latest version.")
			} else {
				fmt.Println("Upload done. Restart now!")
				os.Exit(0)
			}
		} else {
			fmt.Println("Skipping update...")
		}
	}
}
