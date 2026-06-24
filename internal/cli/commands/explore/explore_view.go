/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/explore/explore_view
** File description:
** Rendering of the group explorer screens: groups list, folder tree and local
** file picker, plus the shared status line helper.
 */

package explore

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// View: Render the active screen.
//
// Returns:
// - result1 (string): The rendered view.
func (m exploreModel) View() string {
	switch m.mode {
	case modeGroups:
		return m.viewGroups()
	case modePicker:
		return m.viewPicker()
	default:
		return m.viewTree()
	}
}

// viewGroups: Render the groups selection screen.
//
// Returns:
// - result1 (string): The rendered view.
func (m exploreModel) viewGroups() string {
	out := exploreTitleStyle.Render("flick - your groups") + "\n\n"
	if len(m.groups) == 0 {
		out += exploreCrumbStyle.Render("You don't belong to any group.") + "\n"
	}
	for i, g := range m.groups {
		style := lipgloss.NewStyle()
		if i == m.groupCursor {
			style = style.Bold(true)
		}
		out += "  " + style.Render(fmt.Sprintf("%s (%s)", g.Name, g.Role)) + "\n"
	}
	out += "\n" + exploreHelpStyle.Render("↑/↓ move · → open group · q quit")
	return appendStatus(out, m.status)
}

// viewTree: Render the folder tree screen.
//
// Returns:
// - result1 (string): The rendered view.
func (m exploreModel) viewTree() string {
	out := exploreTitleStyle.Render("flick - "+m.groupName) + "\n\n"
	for i, row := range m.rows {
		name := row.node.name
		if row.node.isFolder {
			name += "/"
		}

		style := lipgloss.NewStyle()
		if row.node.isFolder {
			style = style.Foreground(exploreFolderClr)
		}
		if i == m.cursor {
			style = style.Bold(true)
		}

		out += exploreTreeStyle.Render(row.prefix) + style.Render(name) + "\n"
	}
	if len(m.rows) == 0 && m.status == "" {
		out += exploreCrumbStyle.Render("Empty group.") + "\n"
	}

	if m.creating {
		out += "\n" + "New folder: " + m.nameInput + "▌\n"
		out += exploreHelpStyle.Render("type a name · enter create · esc cancel")
		return appendStatus(out, "")
	}

	help := "↑/↓ move · → open · ← close · d download · esc groups · q quit"
	if m.canManage() {
		help = "↑/↓ move · → open · ← close · d download · u upload · n new folder · x delete · esc groups · q quit"
	}
	out += "\n" + exploreHelpStyle.Render(help)
	return appendStatus(out, m.status)
}

// viewPicker: Render the local file picker screen.
//
// Returns:
// - result1 (string): The rendered view.
func (m exploreModel) viewPicker() string {
	out := exploreTitleStyle.Render("flick - pick files to upload") + "\n"
	out += exploreCrumbStyle.Render(m.pickerDir) + "\n\n"
	for i, item := range m.pickerItems {
		name := item.name
		if item.isDir {
			name += "/"
		}

		style := lipgloss.NewStyle()
		if item.isDir {
			style = style.Foreground(exploreFolderClr)
		}
		if m.pickerSelected[item.path] {
			style = style.Foreground(exploreSelectClr)
		}
		if i == m.pickerCursor {
			style = style.Bold(true)
		}
		out += "  " + style.Render(name) + "\n"
	}
	out += "\n" + exploreHelpStyle.Render("↑/↓ move · → enter dir · ← parent · space select · enter upload · esc cancel")
	return appendStatus(out, m.status)
}

// appendStatus: Append a status or error line to the view when present.
//
// Params:
// - out (string): The view rendered so far.
// - status (string): The status line, empty to append nothing.
//
// Returns:
// - result1 (string): The view with the status line appended.
func appendStatus(out, status string) string {
	if status != "" {
		out += "\n" + status
	}
	return out
}
