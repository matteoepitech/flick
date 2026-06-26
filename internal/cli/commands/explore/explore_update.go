/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/explore/explore_update
** File description:
** Bubble Tea update loop of the group explorer: dispatches messages and routes
** key presses to the handler of the active screen (groups, tree, picker).
 */

package explore

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Init: Start the program by loading the user's groups.
//
// Returns:
// - result1 (tea.Cmd): The initial command.
func (m exploreModel) Init() tea.Cmd {
	return loadGroupsCmd(m.token)
}

// Update: Handle an incoming message and return the next model and command.
//
// Params:
// - msg (tea.Msg): The message to handle.
//
// Returns:
// - result1 (tea.Model): The updated model.
// - result2 (tea.Cmd): The next command, if any.
func (m exploreModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case exploreErrMsg:
		m.status = exploreErrStyle.Render(msg.err.Error())
		return m, nil
	case groupsMsg:
		m.groups = msg.groups
		m.status = ""
		return m, nil
	case folderMsg:
		children := childrenFrom(msg.folders, msg.files)
		m.status = ""
		m.currentID = msg.folderID
		if msg.folderID == "" {
			m.roots = children
			if m.mode != modeTree {
				m.cursor = 0
			}
			m.mode = modeTree
		} else if node := findNode(m.roots, msg.folderID); node != nil {
			node.children = children
			node.loaded = true
			node.expanded = true
		}
		m.rebuild()
		return m, nil
	case actionMsg:
		m.status = msg.note
		if msg.reload {
			return m, loadFolderCmd(m.token, m.groupID, msg.reloadFolder)
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// handleKey: Route a key press to the handler of the active screen.
//
// Params:
// - key (tea.KeyMsg): The key press.
//
// Returns:
// - result1 (tea.Model): The updated model.
// - result2 (tea.Cmd): The next command, if any.
func (m exploreModel) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Folder-name prompt captures keys first.
	if m.creating {
		return m.handleCreating(key)
	}

	switch m.mode {
	case modeGroups:
		return m.handleGroups(key)
	case modeTree:
		return m.handleTree(key)
	case modePicker:
		return m.handlePicker(key)
	}
	return m, nil
}

// handleCreating: Handle a key press while typing a new folder name.
//
// Params:
// - key (tea.KeyMsg): The key press.
//
// Returns:
// - result1 (tea.Model): The updated model.
// - result2 (tea.Cmd): The next command, if any.
func (m exploreModel) handleCreating(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.creating = false
		m.nameInput = ""
	case "enter":
		name := strings.TrimSpace(m.nameInput)
		m.creating = false
		m.nameInput = ""
		if name == "" {
			return m, nil
		}
		parent := m.targetFolder()
		m.status = "Creating..."
		return m, mkdirCmd(m.token, m.groupID, parent, name)
	case "backspace":
		if len(m.nameInput) > 0 {
			m.nameInput = m.nameInput[:len(m.nameInput)-1]
		}
	default:
		if len(key.String()) == 1 {
			m.nameInput += key.String()
		}
	}
	return m, nil
}

// handleGroups: Handle a key press on the groups selection screen.
//
// Params:
// - key (tea.KeyMsg): The key press.
//
// Returns:
// - result1 (tea.Model): The updated model.
// - result2 (tea.Cmd): The next command, if any.
func (m exploreModel) handleGroups(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.groupCursor > 0 {
			m.groupCursor--
		}
	case "down", "j":
		if m.groupCursor < len(m.groups)-1 {
			m.groupCursor++
		}
	case "right", "l", "enter":
		if len(m.groups) == 0 {
			return m, nil
		}
		g := m.groups[m.groupCursor]
		m.groupID, m.groupName, m.groupRole = g.ID, g.Name, g.Role
		m.status = "Loading..."
		return m, loadFolderCmd(m.token, g.ID, "")
	}
	return m, nil
}

// openNode: Open (expanding, lazy-loading if needed) the folder under the cursor.
//
// Returns:
// - result1 (tea.Model): The updated model.
// - result2 (tea.Cmd): The next command, if any.
func (m exploreModel) openNode() (tea.Model, tea.Cmd) {
	if len(m.rows) == 0 {
		return m, nil
	}
	node := m.rows[m.cursor].node
	if !node.isFolder {
		return m, nil
	}
	if !node.loaded {
		m.status = "Loading..."
		return m, loadFolderCmd(m.token, m.groupID, node.id)
	}
	node.expanded = true
	m.rebuild()
	return m, nil
}

// downloadNode: Download the file under the cursor.
//
// Returns:
// - result1 (tea.Model): The updated model.
// - result2 (tea.Cmd): The next command, if any.
func (m exploreModel) downloadNode() (tea.Model, tea.Cmd) {
	if len(m.rows) == 0 {
		return m, nil
	}
	node := m.rows[m.cursor].node
	if node.isFolder {
		return m, nil
	}
	m.status = "Downloading..."
	return m, downloadCmd(m.token, node.code)
}

// returnToGroups: Leave the current group tree and show the group selection.
//
// Returns:
// - result1 (exploreModel): The updated model.
func (m exploreModel) returnToGroups() exploreModel {
	m.mode = modeGroups
	m.roots = nil
	m.rows = nil
	m.cursor = 0
	m.currentID = ""
	m.status = ""
	return m
}

// handleTree: Handle a key press on the folder tree screen.
//
// Params:
// - key (tea.KeyMsg): The key press.
//
// Returns:
// - result1 (tea.Model): The updated model.
// - result2 (tea.Cmd): The next command, if any.
func (m exploreModel) handleTree(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		return m.returnToGroups(), nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
	case "right", "l":
		return m.openNode()
	case "enter":
		// Open the folder under the cursor, or download the file under it.
		if len(m.rows) > 0 && m.rows[m.cursor].node.isFolder {
			return m.openNode()
		}
		return m.downloadNode()
	case "d":
		return m.downloadNode()
	case "left", "h":
		if m.currentID == "" {
			return m.returnToGroups(), nil
		}
		if len(m.rows) == 0 {
			return m, nil
		}
		node := m.rows[m.cursor].node
		if node.isFolder && node.expanded {
			node.expanded = false
			m.rebuild()
		}
	case "u":
		if !m.canManage() {
			m.status = exploreErrStyle.Render("Only a maintainer/owner can upload")
			return m, nil
		}
		dir, err := os.Getwd()
		if err != nil {
			m.status = exploreErrStyle.Render(err.Error())
			return m, nil
		}
		m.pickerTarget = m.targetFolder()
		m.pickerDir = dir
		items, err := loadPicker(dir)
		if err != nil {
			m.status = exploreErrStyle.Render(err.Error())
			return m, nil
		}
		m.pickerItems = items
		m.pickerCursor = 0
		m.pickerSelected = map[string]bool{}
		m.mode = modePicker
	case "n":
		if !m.canManage() {
			m.status = exploreErrStyle.Render("Only a maintainer/owner can create folders")
			return m, nil
		}
		m.creating = true
		m.nameInput = ""
	case "x":
		if !m.canManage() {
			m.status = exploreErrStyle.Render("Only a maintainer/owner can delete")
			return m, nil
		}
		if len(m.rows) == 0 {
			return m, nil
		}
		row := m.rows[m.cursor]
		if row.node.isFolder {
			m.status = "Deleting..."
			return m, rmdirCmd(m.token, m.groupID, row.node.id, row.parentID)
		}
		if row.node.uploadID != "" {
			m.status = "Deleting..."
			return m, rmUploadCmd(m.token, m.groupID, row.node.uploadID, row.parentID)
		}
	}
	return m, nil
}

// handlePicker: Handle a key press on the local file picker screen.
//
// Params:
// - key (tea.KeyMsg): The key press.
//
// Returns:
// - result1 (tea.Model): The updated model.
// - result2 (tea.Cmd): The next command, if any.
func (m exploreModel) handlePicker(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.mode = modeTree
		m.status = ""
	case "up", "k":
		if m.pickerCursor > 0 {
			m.pickerCursor--
		}
	case "down", "j":
		if m.pickerCursor < len(m.pickerItems)-1 {
			m.pickerCursor++
		}
	case "right", "l":
		if len(m.pickerItems) == 0 {
			return m, nil
		}
		item := m.pickerItems[m.pickerCursor]
		if item.isDir {
			if items, err := loadPicker(item.path); err == nil {
				m.pickerDir = item.path
				m.pickerItems = items
				m.pickerCursor = 0
			}
		}
	case "left", "h":
		parent := filepath.Dir(m.pickerDir)
		if parent != m.pickerDir {
			if items, err := loadPicker(parent); err == nil {
				m.pickerDir = parent
				m.pickerItems = items
				m.pickerCursor = 0
			}
		}
	case " ":
		if len(m.pickerItems) > 0 {
			p := m.pickerItems[m.pickerCursor].path
			m.pickerSelected[p] = !m.pickerSelected[p]
		}
	case "enter":
		var srcs []string
		for p, ok := range m.pickerSelected {
			if ok {
				srcs = append(srcs, p)
			}
		}
		if len(srcs) == 0 {
			return m, nil
		}
		m.mode = modeTree
		m.status = "Uploading..."
		return m, uploadCmd(m.token, m.groupID, m.pickerTarget, srcs)
	}
	return m, nil
}
