// Package home implements the Home screen: a list of registered projects with
// status badges and keybindings for refresh, doctor, open.
package home

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TomyQB/mem-cli/internal/registry"
	"github.com/TomyQB/mem-cli/internal/tui/banner"
	"github.com/TomyQB/mem-cli/internal/tui/messages"
	"github.com/TomyQB/mem-cli/internal/tui/styles"
)

// compactListWidth is the inner width available for the bubbles/list inside
// the compact frame.
const compactListWidth = styles.CompactWidth - 10

// Model is the Home screen state.
type Model struct {
	list   list.Model
	reg    *registry.Registry
	width  int
	height int
}

// New creates a Home screen for the given registry.
func New(reg *registry.Registry) Model {
	items := buildItems(reg)
	l := list.New(items, newVioletDelegate(), compactListWidth, listHeight(len(items)))
	l.Title = "Projects"
	l.SetShowTitle(false) // title rendered in the Frame header instead
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	return Model{list: l, reg: reg}
}

// Title satisfies messages.Screen.
func (m Model) Title() string { return "home" }

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles messages for the Home screen.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Compact mode: fixed width, generous height for the list.
		m.list.SetSize(compactListWidth, listHeight(len(m.list.Items())))
		return m, nil
	case messages.RegistryLoadedMsg:
		if msg.Err == nil && msg.Reg != nil {
			m.reg = msg.Reg
			items := buildItems(msg.Reg)
			m.list.SetItems(items)
			m.list.SetSize(compactListWidth, listHeight(len(items)))
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "enter":
			if it, ok := m.list.SelectedItem().(projectItem); ok {
				return m, func() tea.Msg { return messages.ProjectSelectedMsg{Entry: it.entry} }
			}
		case "r":
			return m, messages.LoadRegistryCmd()
		case "d":
			if it, ok := m.list.SelectedItem().(projectItem); ok {
				return m, messages.RunDoctorCmd(it.entry.Path)
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the Home screen inside a compact frame.
func (m Model) View() string {
	header := banner.RenderCompact(styles.CompactInnerWidth, "projects")
	var body string
	if m.reg == nil || len(m.reg.Projects) == 0 {
		body = styles.MenuItem.Render("No projects registered. Run /memcli-init in a repo.")
	} else {
		body = m.list.View()
	}
	footer := styles.FooterHints(
		"j/k", "navigate",
		"enter", "open",
		"r", "refresh",
		"d", "doctor",
		"esc", "back",
	)
	return styles.FrameCompact(header, body, footer, styles.CompactWidth)
}

// listHeight returns a generous height for the list based on item count.
// Each item in the default delegate takes ~2 lines, plus some padding.
func listHeight(n int) int {
	if n <= 0 {
		return 5
	}
	h := n*3 + 2
	if h < 5 {
		h = 5
	}
	if h > 20 {
		h = 20
	}
	return h
}

// newVioletDelegate returns a list.DefaultDelegate re-skinned with the
// deep-violet palette: selected rows in Primary bold, normal rows in Text.
func newVioletDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	primary := lipgloss.Color(styles.ColorPrimary)
	primaryBold := lipgloss.Color(styles.ColorPrimaryBold)
	text := lipgloss.Color(styles.ColorText)
	muted := lipgloss.Color(styles.ColorMuted)
	subtle := lipgloss.Color(styles.ColorSubtle)

	d.Styles.NormalTitle = d.Styles.NormalTitle.Foreground(text)
	d.Styles.NormalDesc = d.Styles.NormalDesc.Foreground(subtle)
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Foreground(primary).
		BorderForeground(primaryBold).
		Bold(true)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.
		Foreground(primary).
		BorderForeground(primaryBold)
	d.Styles.DimmedTitle = d.Styles.DimmedTitle.Foreground(muted)
	d.Styles.DimmedDesc = d.Styles.DimmedDesc.Foreground(muted)
	d.Styles.FilterMatch = d.Styles.FilterMatch.Foreground(primary).Bold(true)
	return d
}

// projectItem is a list.Item wrapping a registry.Entry plus computed status.
type projectItem struct {
	entry  registry.Entry
	status string
}

func (p projectItem) Title() string {
	return fmt.Sprintf("%s %s", styles.Badge(p.status), p.entry.Name)
}

func (p projectItem) Description() string {
	return styles.Dim(p.entry.Path)
}

func (p projectItem) FilterValue() string { return p.entry.Name + " " + p.entry.Path }

func buildItems(reg *registry.Registry) []list.Item {
	if reg == nil {
		return nil
	}
	items := make([]list.Item, 0, len(reg.Projects))
	for _, e := range reg.Projects {
		items = append(items, projectItem{entry: e, status: computeStatus(e)})
	}
	return items
}

// computeStatus is the single source of truth for the Home badge algorithm.
// See design 3.2.
func computeStatus(e registry.Entry) string {
	mem := filepath.Join(e.Path, ".agent-memory")
	info, err := os.Stat(mem)
	if err != nil || !info.IsDir() {
		return styles.StatusMissing
	}
	stale := filepath.Join(mem, ".stale")
	data, err := os.ReadFile(stale)
	if err != nil {
		if os.IsNotExist(err) {
			return styles.StatusOK
		}
		log.Printf("home: cannot read %s: %v", stale, err)
		return styles.StatusOK
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) != "" {
			return styles.StatusStale
		}
	}
	return styles.StatusOK
}
