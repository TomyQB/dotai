package memclitui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/registry"
	"github.com/TomyQB/dotai/internal/tui/banner"
	"github.com/TomyQB/dotai/internal/tui/styles"
)

// entry is one row rendered in the browser.
type entry struct {
	name  string
	abs   string
	isDir bool
}

// BrowserModel walks a project's .agent-memory/ tree with arbitrary depth.
// The current location is tracked by breadcrumbs (slice of subdirectory names
// relative to .agent-memory/). An empty slice means the user is at the root.
type BrowserModel struct {
	project     registry.Entry
	breadcrumbs []string
	items       []entry
	cursor      int
	err         error
	width       int
	height      int
}

// NewBrowser returns a BrowserModel rooted at e.Path/.agent-memory/.
func NewBrowser(e registry.Entry) BrowserModel {
	m := BrowserModel{project: e}
	m.reload()
	return m
}

// Title satisfies Screen.
func (m BrowserModel) Title() string {
	if len(m.breadcrumbs) == 0 {
		return m.project.Name
	}
	return m.project.Name + " / " + strings.Join(m.breadcrumbs, "/")
}

// Init satisfies tea.Model.
func (m BrowserModel) Init() tea.Cmd { return nil }

// Update handles keyboard input.
func (m BrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter", "right":
			return m.enterSelected()
		case "left":
			return m.goBack()
		case "esc":
			return m, popCmd()
		}
	}
	return m, nil
}

// enterSelected drills into a directory or opens a file in the preview screen.
func (m BrowserModel) enterSelected() (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.items) {
		return m, nil
	}
	sel := m.items[m.cursor]
	if sel.isDir {
		m.breadcrumbs = append(m.breadcrumbs, sel.name)
		m.reload()
		return m, nil
	}
	return m, pushCmd(NewPreview(sel.abs))
}

// goBack climbs one level up. When at the root of the project, it pops the
// whole browser screen so the user returns to the projects list.
func (m BrowserModel) goBack() (tea.Model, tea.Cmd) {
	if len(m.breadcrumbs) == 0 {
		return m, popCmd()
	}
	m.breadcrumbs = m.breadcrumbs[:len(m.breadcrumbs)-1]
	m.reload()
	return m, nil
}

// reload rebuilds the item list from the current breadcrumb path and resets
// the cursor to the top.
func (m *BrowserModel) reload() {
	m.cursor = 0
	abs := m.currentAbs()
	raw, err := os.ReadDir(abs)
	if err != nil {
		m.items = nil
		m.err = err
		return
	}
	m.err = nil
	var dirs, files []entry
	for _, d := range raw {
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		e := entry{
			name:  name,
			abs:   filepath.Join(abs, name),
			isDir: d.IsDir(),
		}
		if e.isDir {
			dirs = append(dirs, e)
		} else {
			files = append(files, e)
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].name < dirs[j].name })
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	m.items = append(dirs, files...)
}

// currentAbs returns the absolute path of the directory currently displayed.
func (m BrowserModel) currentAbs() string {
	parts := append([]string{m.project.Path, ".agent-memory"}, m.breadcrumbs...)
	return filepath.Join(parts...)
}

// SetSize stores terminal dimensions.
func (m *BrowserModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// View renders the browser as a compact frame with a breadcrumb header.
func (m BrowserModel) View() string {
	screen := m.project.Name
	if len(m.breadcrumbs) > 0 {
		screen += " / " + strings.Join(m.breadcrumbs, "/")
	}
	header := banner.RenderCompact(styles.CompactInnerWidth, screen)

	var body strings.Builder
	crumb := ".agent-memory/"
	if len(m.breadcrumbs) > 0 {
		crumb += strings.Join(m.breadcrumbs, "/") + "/"
	}
	body.WriteString(styles.SectionTitle.Render(crumb))
	body.WriteString("\n\n")

	if m.err != nil {
		body.WriteString(styles.ProgressError.Render("Error: " + m.err.Error()))
	} else if len(m.items) == 0 {
		body.WriteString(styles.Help.Render("(empty)"))
	} else {
		for i, e := range m.items {
			prefix := "  "
			if i == m.cursor {
				prefix = styles.Arrow.Render("▸ ")
			}
			label := e.name
			if e.isDir {
				label += "/"
				if i == m.cursor {
					body.WriteString(prefix + styles.SectionTitle.Render(label))
				} else {
					body.WriteString("  " + styles.SectionTitle.Render(label))
				}
			} else {
				if i == m.cursor {
					body.WriteString(prefix + styles.MenuItemSelected.Render(label))
				} else {
					body.WriteString("  " + styles.MenuItem.Render(label))
				}
			}
			body.WriteString("\n")
		}
	}

	footer := styles.FooterHints(
		"↑/↓", "navigate",
		"enter/→", "open",
		"←", "up",
		"esc", "back",
	)
	return styles.FrameCompact(header, body.String(), footer, styles.CompactWidth)
}
