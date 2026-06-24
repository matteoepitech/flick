/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/explore/explore_commands
** File description:
** Asynchronous Bubble Tea commands of the group explorer: each one performs a
** network call and reports its result through a message.
 */

package explore

import tea "github.com/charmbracelet/bubbletea"

// loadGroupsCmd: Build the command that loads the user's groups.
//
// Params:
// - token (string): The session token.
//
// Returns:
// - result1 (tea.Cmd): A command emitting groupsMsg or exploreErrMsg.
func loadGroupsCmd(token string) tea.Cmd {
	return func() tea.Msg {
		groups, err := fetchMyGroups(token)
		if err != nil {
			return exploreErrMsg{err}
		}
		return groupsMsg{groups}
	}
}

// loadFolderCmd: Build the command that loads one folder level of a group.
//
// Params:
// - token (string): The session token.
// - groupID (string): The group to explore.
// - folderID (string): The folder level, or "" for the group root.
//
// Returns:
// - result1 (tea.Cmd): A command emitting folderMsg or exploreErrMsg.
func loadFolderCmd(token, groupID, folderID string) tea.Cmd {
	return func() tea.Msg {
		folders, files, err := fetchExplore(token, groupID, folderID)
		if err != nil {
			return exploreErrMsg{err}
		}
		return folderMsg{folderID: folderID, folders: folders, files: files}
	}
}

// downloadCmd: Build the command that downloads a file by its share code.
//
// Params:
// - token (string): The session token.
// - code (string): The transfer's share code.
//
// Returns:
// - result1 (tea.Cmd): A command emitting actionMsg or exploreErrMsg.
func downloadCmd(token, code string) tea.Cmd {
	return func() tea.Msg {
		name, err := downloadGroupFile(token, code)
		if err != nil {
			return exploreErrMsg{err}
		}
		return actionMsg{note: "Downloaded " + name}
	}
}

// uploadCmd: Build the command that uploads local sources into a group folder.
//
// Params:
// - token (string): The session token.
// - groupID (string): The target group.
// - folderID (string): The destination folder, or "" for the root.
// - srcs ([]string): The local paths to upload.
//
// Returns:
// - result1 (tea.Cmd): A command emitting actionMsg or exploreErrMsg.
func uploadCmd(token, groupID, folderID string, srcs []string) tea.Cmd {
	return func() tea.Msg {
		if err := uploadToGroupFolder(token, groupID, folderID, srcs); err != nil {
			return exploreErrMsg{err}
		}
		return actionMsg{note: "Uploaded", reload: true, reloadFolder: folderID}
	}
}

// mkdirCmd: Build the command that creates a folder in a group.
//
// Params:
// - token (string): The session token.
// - groupID (string): The target group.
// - parentID (string): The parent folder id, or "" for the root.
// - name (string): The new folder name.
//
// Returns:
// - result1 (tea.Cmd): A command emitting actionMsg or exploreErrMsg.
func mkdirCmd(token, groupID, parentID, name string) tea.Cmd {
	return func() tea.Msg {
		if err := createGroupFolder(token, groupID, parentID, name); err != nil {
			return exploreErrMsg{err}
		}
		return actionMsg{note: "Folder created", reload: true, reloadFolder: parentID}
	}
}

// rmdirCmd: Build the command that deletes a folder from a group.
//
// Params:
// - token (string): The session token.
// - groupID (string): The target group.
// - folderID (string): The folder to delete.
// - parentID (string): The deleted folder's parent, reloaded afterwards.
//
// Returns:
// - result1 (tea.Cmd): A command emitting actionMsg or exploreErrMsg.
func rmdirCmd(token, groupID, folderID, parentID string) tea.Cmd {
	return func() tea.Msg {
		if err := deleteGroupFolder(token, groupID, folderID); err != nil {
			return exploreErrMsg{err}
		}
		return actionMsg{note: "Folder deleted", reload: true, reloadFolder: parentID}
	}
}
