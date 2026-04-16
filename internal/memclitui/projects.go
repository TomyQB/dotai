package memclitui

import (
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/registry"
	"github.com/TomyQB/dotai/internal/tui/banner"
	"github.com/TomyQB/dotai/internal/tui/styles"
)

// ProjectsModel is the landing screen: a scrollable list of registered
// projects pulled from the memcli registry. Entering one pushes a browser
// rooted at its .agent-memory/ directory.
type ProjectsModel struct {
	reg    *registry.Registry
	rows   []registry.Entry
	cursor int
	width  int
	height int
}

// NewProjects builds a ProjectsModel from the given registry. Entries whose
// on-disk path has disappeared are filtered out so the list never shows
// ghost rows (the underlying registry file is not mutated).
func NewProjects(reg *registry.Registry) ProjectsModel {
	rows := make([]registry.Entry, 0, len(reg.Projects))
	for _, e := range reg.Projects {
		if _, err := os.Stat(e.Path); err == nil {
			rows = append(rows, e)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].LastSeen.After(rows[j].LastSeen) })
	return ProjectsModel{reg: reg, rows: rows}
}

// Title satisfies Screen.
func (m ProjectsModel) Title() string { return "projects" }

// Init satisfies tea.Model.
func (m ProjectsModel) Init() tea.Cmd { return nil }

// Update handles keyboard input.
func (m ProjectsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
		case "enter", "right":
			if m.cursor < len(m.rows) {
				return m, pushCmd(NewBrowser(m.rows[m.cursor]))
			}
		case "esc", "q", "left":
			return m, tea.Quit
		}
	}
	return m, nil
}

// SetSize stores terminal dimensions.
func (m *ProjectsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// View renders the projects list in a compact frame.
func (m ProjectsModel) View() string {
	header := banner.RenderCompact(styles.CompactInnerWidth, "projects")

	var body strings.Builder
	body.WriteString(styles.SectionTitle.Render("Registered projects"))
	body.WriteString("\n\n")
	if len(m.rows) == 0 {
		body.WriteString(styles.Help.Render("No projects registered yet."))
		body.WriteString("\n")
		body.WriteString(styles.Help.Render("Run `/memcli-init` from Claude Code inside a repo to add one."))
	} else {
		for i, e := range m.rows {
			prefix := "  "
			if i == m.cursor {
				prefix = styles.Arrow.Render("▸ ")
			}
			var name string
			if i == m.cursor {
				name = styles.MenuItemSelected.Render(e.Name)
			} else {
				name = styles.MenuItem.Render(e.Name)
			}
			body.WriteString(prefix + name + "  " + styles.Dim(e.Path))
			body.WriteString("\n")
		}
	}

	footer := styles.FooterHints(
		"↑/↓", "navigate",
		"enter/→", "open",
		"q/esc", "quit",
	)
	return styles.FrameCompact(header, body.String(), footer, styles.CompactWidth)
}
