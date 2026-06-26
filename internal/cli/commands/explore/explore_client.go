/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/explore/explore_client
** File description:
** Bearer-authenticated HTTP client for the interactive group explorer.
 */

package explore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/Flick-Corp/flick/internal/cli/config"
	"github.com/Flick-Corp/flick/internal/cli/network"
	archiveutil "github.com/Flick-Corp/flick/internal/utils/archive"
	"github.com/Flick-Corp/flick/internal/utils/checksum"
	tusutil "github.com/Flick-Corp/flick/internal/utils/tus"
	tus "github.com/eventials/go-tus"
)

// exploreGroup: a group the user belongs to, as shown on the groups screen.
type exploreGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

// exploreFolder: a sub-folder at the explored level.
type exploreFolder struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// exploreFile: a transfer at the explored level, with its real (resolved) name.
// The code stays internal: it is only used to download through the native route.
type exploreFile struct {
	id   string
	name string
	code string
}

// fetchMyGroups: Resolve the groups the user belongs to from /whoami.
//
// Params:
// - token (string): The session token.
//
// Returns:
// - result1 ([]exploreGroup): The user's groups.
// - result2 (error): An error if the call failed.
func fetchMyGroups(token string) ([]exploreGroup, error) {
	body, err := json.Marshal(map[string]string{"token": token})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, config.Conf.APIBaseURL()+"/whoami", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := network.SharedClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach the server: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %s", resp.Status)
	}

	var result struct {
		User struct {
			Groups []exploreGroup `json:"groups"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.User.Groups, nil
}

// authedGet: GET a group route with the Bearer token and decode the JSON body.
//
// Params:
// - token (string): The session token.
// - url (string): The full URL to GET.
// - out (any): The destination to decode the JSON response into.
//
// Returns:
// - result1 (error): An error if the call failed or returned a non-200 status.
func authedGet(token, url string, out any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := network.SharedClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach the server: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// fetchExplore: List the sub-folders and files at a level of a group's tree
// (the root when folderID is empty), resolving each file's real name.
//
// Params:
// - token (string): The session token.
// - groupID (string): The group to explore.
// - folderID (string): The folder level, or "" for the group root.
//
// Returns:
// - result1 ([]exploreFolder): The sub-folders at this level.
// - result2 ([]exploreFile): The files at this level, with real names.
// - result3 (error): An error if the call failed.
func fetchExplore(token, groupID, folderID string) ([]exploreFolder, []exploreFile, error) {
	url := config.Conf.APIBaseURL() + "/admin/groups/" + groupID + "/explore"
	if folderID != "" {
		url += "?folder=" + folderID
	}

	var res struct {
		Folders []exploreFolder `json:"folders"`
		Uploads []struct {
			ID   string `json:"id"`
			Code string `json:"code"`
		} `json:"uploads"`
	}
	if err := authedGet(token, url, &res); err != nil {
		return nil, nil, err
	}

	files := make([]exploreFile, 0, len(res.Uploads))
	for _, upload := range res.Uploads {
		files = append(files, exploreFile{
			id:   upload.ID,
			name: resolveUploadName(token, upload.Code),
			code: upload.Code,
		})
	}
	return res.Folders, files, nil
}

// resolveUploadName: Resolve a transfer's real file name(s) from /download/info,
// falling back to the code if the listing cannot be read.
//
// Params:
// - token (string): The session token.
// - code (string): The transfer's share code.
//
// Returns:
// - result1 (string): The joined item names, or the code on failure.
func resolveUploadName(token, code string) string {
	var info struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := authedGet(token, config.Conf.APIBaseURL()+"/download/info?code="+code, &info); err != nil {
		return code
	}

	names := make([]string, 0, len(info.Items))
	for _, item := range info.Items {
		names = append(names, item.Name)
	}
	if len(names) == 0 {
		return code
	}
	return strings.Join(names, ", ")
}

// createGroupFolder: Create a folder in the group (maintainer/owner). An empty
// parentID creates it at the group root.
//
// Params:
// - token (string): The session token.
// - groupID (string): The target group.
// - parentID (string): The parent folder id, or "" for the root.
// - name (string): The folder name.
//
// Returns:
// - result1 (error): An error if the call failed.
func createGroupFolder(token, groupID, parentID, name string) error {
	body, err := json.Marshal(map[string]string{"name": name, "parent_id": parentID})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, config.Conf.APIBaseURL()+"/admin/groups/"+groupID+"/folders", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := network.SharedClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach the server: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("server returned %s", resp.Status)
	}
	return nil
}

// deleteGroupFolder: Delete a folder and its contents (maintainer/owner).
//
// Params:
// - token (string): The session token.
// - groupID (string): The target group.
// - folderID (string): The folder to delete.
//
// Returns:
// - result1 (error): An error if the call failed.
func deleteGroupFolder(token, groupID, folderID string) error {
	req, err := http.NewRequest(http.MethodDelete, config.Conf.APIBaseURL()+"/admin/groups/"+groupID+"/folders/"+folderID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := network.SharedClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach the server: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("server returned %s", resp.Status)
	}
	return nil
}

// deleteGroupUpload: Revoke a file shared with a group (maintainer/owner).
//
// Params:
// - token (string): The session token.
// - groupID (string): The target group.
// - uploadID (string): The group upload row to delete.
//
// Returns:
// - result1 (error): An error if the call failed.
func deleteGroupUpload(token, groupID, uploadID string) error {
	req, err := http.NewRequest(http.MethodDelete, config.Conf.APIBaseURL()+"/admin/groups/"+groupID+"/uploads/"+uploadID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := network.SharedClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach the server: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("server returned %s", resp.Status)
	}
	return nil
}

// uploadToGroupFolder: Archive the local sources and stream them into a group
// folder with the tus resumable protocol (Bearer auth, group_id/folder_id passed
// as upload metadata). An empty folderID uploads at the group root. Silent (no
// progress bar) so it can run under the TUI.
//
// Params:
// - token (string): The session token.
// - groupID (string): The target group.
// - folderID (string): The destination folder, or "" for the root.
// - srcs ([]string): The local paths to upload.
//
// Returns:
// - result1 (error): An error if the upload failed.
func uploadToGroupFolder(token, groupID, folderID string, srcs []string) error {
	archivePath, err := archiveutil.ToTemp(srcs, nil)
	if err != nil {
		return err
	}
	defer os.Remove(archivePath)

	sum, err := checksum.HashFile(archivePath)
	if err != nil {
		return fmt.Errorf("cannot checksum the archive: %w", err)
	}

	archive, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()

	stat, err := archive.Stat()
	if err != nil {
		return err
	}

	metadata := tus.Metadata{
		"filename":   archiveutil.RandomName(),
		"checksum":   sum,
		"encrypted":  "false",
		"expiration": config.Conf.DefExpTime,
		"groupId":    groupID,
	}
	if folderID != "" {
		metadata["folderId"] = folderID
	}

	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)

	client, err := tus.NewClient(config.Conf.APIBaseURL()+"/upload/", &tus.Config{
		ChunkSize:  tusutil.ChunkSize,
		HttpClient: network.SharedClient,
		Header:     header,
	})
	if err != nil {
		return fmt.Errorf("cannot create the upload client: %w", err)
	}

	upload := tus.NewUpload(archive, stat.Size(), metadata, "")
	uploader, err := client.CreateUpload(upload)
	if err != nil {
		return fmt.Errorf("cannot start the upload: %w", err)
	}
	if err := uploader.Upload(); err != nil {
		return fmt.Errorf("the upload failed: %w", err)
	}
	return nil
}

// downloadGroupFile: Download a group transfer by code through the native
// /download endpoint (Bearer) and extract each archive part into the current
// directory, exactly like the standalone download command. Silent so it can run
// under the TUI.
//
// Params:
// - token (string): The session token.
// - code (string): The transfer's share code.
//
// Returns:
// - result1 (string): The extracted top-level name(s) on success.
// - result2 (error): An error if the download failed.
func downloadGroupFile(token, code string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, config.Conf.APIBaseURL()+"/download?code="+code, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := network.SharedClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to reach the server: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned %s", resp.Status)
	}

	_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return "", fmt.Errorf("invalid response: %w", err)
	}
	reader := multipart.NewReader(resp.Body, params["boundary"])

	names := make([]string, 0)
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if part.FormName() != "file" {
			part.Close()
			continue
		}

		extracted, err := extractPart(part)
		part.Close()
		if err != nil {
			return "", err
		}
		names = append(names, extracted...)
	}

	if len(names) == 0 {
		return code, nil
	}
	return strings.Join(names, ", "), nil
}

// extractPart: Buffer one archive part to a temp file and extract it into the
// current directory, returning its top-level entry names. Group transfers are
// never encrypted, so no decryption step is needed here.
//
// Params:
// - part (io.Reader): The multipart file stream carrying the zip archive.
//
// Returns:
// - result1 ([]string): The unique top-level names extracted.
// - result2 (error): An error if the part could not be buffered or extracted.
func extractPart(part io.Reader) ([]string, error) {
	tmp, err := os.CreateTemp("", "flick-download-*.zip")
	if err != nil {
		return nil, fmt.Errorf("cannot create temp archive: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := io.Copy(tmp, part); err != nil {
		tmp.Close()
		return nil, fmt.Errorf("cannot download the archive: %w", err)
	}
	tmp.Close()

	return archiveutil.Extract(tmp.Name(), ".")
}
