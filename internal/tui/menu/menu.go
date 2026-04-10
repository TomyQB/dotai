// Package menu implements the root landing screen of the TUI: an ASCII banner
// with a short list of top-level actions. Rendering is done by hand (no
// bubbles/list) so we have exact control over selection chrome and colors.
package menu

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/mem-cli/internal/tui/banner"
	"github.com/TomyQB/mem-cli/internal/tui/styles"
)

// Action identifies which top-level operation the user picked.
type Action int

const (
	ActionNone Action = iota
	ActionInstall
	ActionBrowseProjects
	ActionDoctorCWD
	ActionPruneRegistry
	ActionQuit
)

// SelectedMsg is emitted when the user presses enter on a menu item.
type SelectedMsg struct{ Action Action }

// ToastMsg sets a transient status line shown under the menu.
type ToastMsg struct{ Text string }

// clearToastMsg clears the transient status line.
type clearToastMsg struct{}

// item is one row on the menu.
type item struct {
	label  string
	action Action
}

var items = []item{
	{"Install / Reinstall", ActionInstall},
	{"Browse projects", ActionBrowseProjects},
	{"Doctor (current directory)", ActionDoctorCWD},
	{"Prune registry", ActionPruneRegistry},
	{"Quit", ActionQuit},
}

// Model is the menu screen state.
type Model struct {
	cursor int
	width  int
	height int
	toast  string
}

// New creates a Menu model with the cursor on the first item.
func New() Model { return Model{} }

// Title satisfies messages.Screen.
func (m Model) Title() string { return "menu" }

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles messages for the Menu screen.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case ToastMsg:
		m.toast = msg.Text
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearToastMsg{} })
	case clearToastMsg:
		m.toast = ""
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(items)-1 {
				m.cursor++
			}
			return m, nil
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "enter":
			action := items[m.cursor].action
			if action == ActionQuit {
				return m, tea.Quit
			}
			return m, func() tea.Msg { return SelectedMsg{Action: action} }
		}
	}
	return m, nil
}

// View renders the Menu screen using FrameCompact (no alt-screen stretching).
func (m Model) View() string {
	header := banner.Render(styles.CompactInnerWidth)
	body := m.renderBody()
	footer := styles.FooterHints(
		"j/k", "navigate",
		"enter", "select",
		"q", "quit",
	)
	return styles.FrameCompact(header, body, footer, styles.CompactWidth)
}

func (m Model) renderBody() string {
	var b strings.Builder
	b.WriteString(styles.SectionTitle.Render("Menu"))
	b.WriteString("\n\n")
	for i, it := range items {
		if i == m.cursor {
			b.WriteString("  ")
			b.WriteString(styles.Arrow.Render("▸ "))
			b.WriteString(styles.MenuItemSelected.Render(it.label))
		} else {
			b.WriteString("    ")
			b.WriteString(styles.MenuItem.Render(it.label))
		}
		b.WriteString("\n")
	}
	if m.toast != "" {
		b.WriteString("\n")
		b.WriteString(styles.Toast.Render(m.toast))
		b.WriteString("\n")
	}
	return b.String()
}
