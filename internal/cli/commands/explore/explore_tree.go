/*
** FLICK PROJECT, 2026
** flick/internal/cli/commands/explore/explore_tree
** File description:
** Tree and picker helpers of the group explorer: build nodes from a level,
** locate a node, flatten the tree into rows and list a local directory.
 */

package explore

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// childrenFrom: Build tree nodes from a level's folders and files.
//
// Params:
// - folders ([]exploreFolder): The sub-folders at this level.
// - files ([]exploreFile): The files at this level.
//
// Returns:
// - result1 ([]*exploreNode): The folder and file nodes, folders first.
func childrenFrom(folders []exploreFolder, files []exploreFile) []*exploreNode {
	out := make([]*exploreNode, 0, len(folders)+len(files))
	for _, f := range folders {
		out = append(out, &exploreNode{id: f.ID, name: f.Name, isFolder: true})
	}
	for _, f := range files {
		out = append(out, &exploreNode{id: f.id, name: f.name, uploadID: f.id, code: f.code})
	}
	return out
}

// findNode: Find the folder node carrying the given id in the tree.
//
// Params:
// - nodes ([]*exploreNode): The nodes to search, recursively.
// - id (string): The folder id to look for.
//
// Returns:
// - result1 (*exploreNode): The matching node, or nil when not found.
func findNode(nodes []*exploreNode, id string) *exploreNode {
	for _, n := range nodes {
		if n.isFolder && n.id == id {
			return n
		}
		if found := findNode(n.children, id); found != nil {
			return found
		}
	}
	return nil
}

// rebuild: Flatten the expanded tree into the visible rows and clamp the cursor.
func (m *exploreModel) rebuild() {
	m.rows = nil
	var walk func(nodes []*exploreNode, ancestorsLast []bool, parentID string)
	walk = func(nodes []*exploreNode, ancestorsLast []bool, parentID string) {
		for i, n := range nodes {
			last := i == len(nodes)-1

			var prefix strings.Builder
			for _, parentLast := range ancestorsLast {
				if parentLast {
					prefix.WriteString("    ")
				} else {
					prefix.WriteString("│   ")
				}
			}
			if last {
				prefix.WriteString("└── ")
			} else {
				prefix.WriteString("├── ")
			}

			m.rows = append(m.rows, exploreRow{node: n, prefix: prefix.String(), parentID: parentID})
			if n.isFolder && n.expanded {
				walk(n.children, append(ancestorsLast, last), n.id)
			}
		}
	}
	walk(m.roots, nil, "")

	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// targetFolder: The group folder an action applies to. It uses the folder level
// that contains the highlighted row, except when the highlighted row is the
// currently opened folder itself (notably an empty folder).
//
// Returns:
// - result1 (string): The target folder id, or "" for the group root.
func (m exploreModel) targetFolder() string {
	if len(m.rows) == 0 {
		return m.currentID
	}

	row := m.rows[m.cursor]
	if row.node.isFolder && row.node.id == m.currentID {
		return row.node.id
	}
	return row.parentID
}

// loadPicker: List the visible entries of a local directory for the picker.
//
// Params:
// - dir (string): The directory to read.
//
// Returns:
// - result1 ([]pickerItem): The directory entries, folders first then by name.
// - result2 (error): An error if the directory cannot be read.
func loadPicker(dir string) ([]pickerItem, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	items := make([]pickerItem, 0, len(entries))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		items = append(items, pickerItem{name: e.Name(), path: filepath.Join(dir, e.Name()), isDir: e.IsDir()})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].isDir != items[j].isDir {
			return items[i].isDir
		}
		return items[i].name < items[j].name
	})
	return items, nil
}
