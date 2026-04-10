// Package project implements the Project screen: a two-level browser for a
// project's .agent-memory/ directory, with stale markers sourced from .stale.
//
// Level 0 (root): shows root-level folders and files.
// Level 1 (folder): shows files inside the selected folder.
package project

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/mem-cli/internal/registry"
	"github.com/TomyQB/mem-cli/internal/tui/banner"
	"github.com/TomyQB/mem-cli/internal/tui/messages"
	"github.com/TomyQB/mem-cli/internal/tui/styles"
)

// entry represents one row in the project browser.
type entry struct {
	name  string // display name (e.g. "flows/", "index.md")
	rel   string // relative path from .agent-memory/
	abs   string // absolute path
	isDir bool
	stale bool
}

// Model is the Project screen state.
type Model struct {
	items    []entry
	cursor   int
	entry    registry.Entry
	staleSet map[string]struct{}
	depth    int    // 0 = root, 1 = inside folder
	folder   string // current folder name when depth=1
	width    int
	height   int
}

// New creates a Project screen for the given entry.
func New(e registry.Entry) Model {
	root := filepath.Join(e.Path, ".agent-memory")
	staleSet := loadStaleSet(filepath.Join(root, ".stale"))
	m := Model{
		entry:    e,
		staleSet: staleSet,
	}
	m.items = m.buildRootItems()
	return m
}

// Title satisfies messages.Screen.
func (m Model) Title() string { return m.entry.Name }

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
			return m, nil
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "enter":
			if m.cursor >= len(m.items) {
				return m, nil
			}
			selected := m.items[m.cursor]
			if selected.isDir {
				// Drill into folder (depth 0 → 1).
				m.depth = 1
				m.folder = selected.name
				m.items = m.buildFolderItems(selected.rel)
				m.cursor = 0
				return m, nil
			}
			// Open file in preview.
			path := selected.abs
			return m, func() tea.Msg { return messages.DocSelectedMsg{Path: path} }
		case "esc":
			if m.depth == 1 {
				// Go back to root level.
				m.depth = 0
				m.folder = ""
				m.items = m.buildRootItems()
				m.cursor = 0
				return m, nil
			}
			// At root: let the parent handle esc (pop screen).
			return m, func() tea.Msg { return messages.PopScreenMsg{} }
		}
	}
	return m, nil
}

// View renders the Project screen inside a compact frame.
func (m Model) View() string {
	screenName := m.entry.Name
	if m.depth == 1 {
		screenName += " / " + m.folder
	}
	header := banner.RenderCompact(styles.CompactInnerWidth, screenName)
	body := m.renderBody()
	footer := styles.FooterHints(
		"j/k", "navigate",
		"enter", "open",
		"esc", "back",
	)
	return styles.FrameCompact(header, body, footer, styles.CompactWidth)
}

func (m Model) renderBody() string {
	if len(m.items) == 0 {
		return styles.MenuItem.Render("(empty)")
	}
	var b strings.Builder
	label := ".agent-memory/"
	if m.depth == 1 {
		label = m.folder + "/"
	}
	b.WriteString(styles.SectionTitle.Render(label))
	b.WriteString("\n\n")
	for i, e := range m.items {
		if i == m.cursor {
			b.WriteString("  ")
			b.WriteString(styles.Arrow.Render("▸ "))
			b.WriteString(m.renderEntry(e, true))
		} else {
			b.WriteString("    ")
			b.WriteString(m.renderEntry(e, false))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// renderEntry renders one line for the given entry.
func (m Model) renderEntry(e entry, selected bool) string {
	name := e.name
	if e.isDir {
		name += "/"
	}

	var parts []string
	if e.stale {
		parts = append(parts, styles.BadgeStale())
	}
	if selected {
		if e.isDir {
			parts = append(parts, styles.SectionTitle.Render(name))
		} else {
			parts = append(parts, styles.MenuItemSelected.Render(name))
		}
	} else {
		if e.isDir {
			parts = append(parts, styles.SectionTitle.Render(name))
		} else {
			parts = append(parts, styles.MenuItem.Render(name))
		}
	}
	return strings.Join(parts, " ")
}

// buildRootItems lists root-level entries in .agent-memory/.
func (m Model) buildRootItems() []entry {
	root := filepath.Join(m.entry.Path, ".agent-memory")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var dirs, files []entry
	for _, d := range entries {
		name := d.Name()
		// Skip hidden entries except .stale.
		if strings.HasPrefix(name, ".") && name != ".stale" {
			continue
		}
		abs := filepath.Join(root, name)
		_, stale := m.staleSet[name]
		e := entry{
			name:  name,
			rel:   name,
			abs:   abs,
			isDir: d.IsDir(),
			stale: stale,
		}
		if d.IsDir() {
			dirs = append(dirs, e)
		} else {
			files = append(files, e)
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].name < dirs[j].name })
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	return append(dirs, files...)
}

// buildFolderItems lists files inside a specific folder in .agent-memory/.
func (m Model) buildFolderItems(folderRel string) []entry {
	root := filepath.Join(m.entry.Path, ".agent-memory")
	folderAbs := filepath.Join(root, folderRel)
	entries, err := os.ReadDir(folderAbs)
	if err != nil {
		return nil
	}
	var items []entry
	for _, d := range entries {
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		// Only show files at depth 1 (no recursive nesting).
		if d.IsDir() {
			continue
		}
		rel := filepath.Join(folderRel, name)
		abs := filepath.Join(folderAbs, name)
		_, stale := m.staleSet[rel]
		items = append(items, entry{
			name:  name,
			rel:   rel,
			abs:   abs,
			isDir: false,
			stale: stale,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].name < items[j].name })
	return items
}

func loadStaleSet(path string) map[string]struct{} {
	set := make(map[string]struct{})
	data, err := os.ReadFile(path)
	if err != nil {
		return set
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			set[trimmed] = struct{}{}
		}
	}
	return set
}
