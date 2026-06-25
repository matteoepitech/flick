/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/explore/explore_model
** File description:
** State of the interactive group explorer: screen modes, styles, tree and
** picker types, and the Bubble Tea model with its asynchronous messages.
 */

package explore

import "github.com/charmbracelet/lipgloss"

// Explorer screens, used as exploreModel.mode values.
const (
	modeGroups = 0 // group selection screen
	modeTree   = 1 // folder tree of the selected group
	modePicker = 2 // local file picker used to choose uploads
)

// Styles.
var (
	exploreTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	exploreCrumbStyle = lipgloss.NewStyle().Faint(true)
	exploreTreeStyle  = lipgloss.NewStyle().Faint(true)
	exploreFolderClr  = lipgloss.Color("39")  // blue for folders
	exploreSelectClr  = lipgloss.Color("208") // orange for selected items
	exploreHelpStyle  = lipgloss.NewStyle().Faint(true)
	exploreErrStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

// exploreNode: a node in the lazily-loaded tree. Folders carry children once
// loaded; files carry the share code used to download them.
type exploreNode struct {
	id       string
	name     string
	isFolder bool
	code     string
	loaded   bool
	expanded bool
	children []*exploreNode
}

// exploreRow: one visible line of the flattened tree.
type exploreRow struct {
	node     *exploreNode
	prefix   string
	parentID string // folder id this row sits in ("" = group root)
}

// pickerItem: one entry of the local file picker.
type pickerItem struct {
	name  string
	path  string
	isDir bool
}

// exploreModel: the Bubble Tea state.
type exploreModel struct {
	token  string
	mode   int
	status string

	groups      []exploreGroup
	groupCursor int

	groupID   string
	groupName string
	groupRole string
	roots     []*exploreNode
	rows      []exploreRow
	cursor    int
	currentID string // folder whose contents receive create/upload actions

	// folder creation prompt (tree mode)
	creating  bool
	nameInput string

	// local file picker (upload)
	pickerDir      string
	pickerItems    []pickerItem
	pickerCursor   int
	pickerSelected map[string]bool
	pickerTarget   string // group folder id to upload into
}

// groupsMsg: carries the user's groups once loaded.
type groupsMsg struct {
	groups []exploreGroup
}

// folderMsg: carries the folders and files of one explored level.
type folderMsg struct {
	folderID string
	folders  []exploreFolder
	files    []exploreFile
}

// actionMsg: carries the outcome of an action, optionally asking for a reload.
type actionMsg struct {
	note         string
	reload       bool
	reloadFolder string
}

// exploreErrMsg: carries an error to be shown in the status line.
type exploreErrMsg struct {
	err error
}

// canManage: Whether the current user may create/delete folders and upload.
//
// Returns:
// - result1 (bool): True when the user is a maintainer or an owner.
func (m exploreModel) canManage() bool {
	return m.groupRole == "maintainer" || m.groupRole == "owner"
}
