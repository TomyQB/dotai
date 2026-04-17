// Package localinstall implements the Install screen for `dotai local`.
// The screen shows every profile as a checkbox row; confirming applies the
// selected profiles to targetDir/.claude/ (skills and optional CLAUDE.md).
package localinstall

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/localprofile"
	"github.com/TomyQB/dotai/internal/tui/banner"
	"github.com/TomyQB/dotai/internal/tui/messages"
	"github.com/TomyQB/dotai/internal/tui/styles"
)

// phase tracks whether the screen is interactive, running, or done.
type phase int

const (
	phaseIdle phase = iota
	phaseRunning
	phaseDone
)

// profileRow pairs a profile with its selection state and installed flag.
type profileRow struct {
	profile   localprofile.Profile
	selected  bool
	installed bool
}

// doneMsg is emitted once install work finishes (or fails).
type doneMsg struct {
	successes []string
	err       error
}

// Model is the Bubble Tea model for the local Install screen.
type Model struct {
	targetDir     string
	rows          []profileRow
	cursor        int
	phase         phase
	toast         string
	toastErr      bool
	width, height int
}

// New returns an Install screen bound to targetDir. Profiles already present
// in targetDir are pre-checked so Enter acts as "reinstall" for them.
func New(targetDir string) Model {
	profiles := localprofile.All()
	rows := make([]profileRow, len(profiles))
	for i, p := range profiles {
		installed := p.IsInstalled(targetDir)
		rows[i] = profileRow{profile: p, selected: !installed, installed: installed}
	}
	return Model{targetDir: targetDir, rows: rows}
}

// Title satisfies messages.Screen.
func (m Model) Title() string { return "local install" }

// Init has no startup work.
func (m Model) Init() tea.Cmd { return nil }

// Update handles key events and install-completion messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case doneMsg:
		m.phase = phaseDone
		if msg.err != nil {
			m.toast = "install failed: " + msg.err.Error()
			m.toastErr = true
		} else if len(msg.successes) == 0 {
			m.toast = "no profiles selected"
		} else {
			m.toast = "installed: " + strings.Join(msg.successes, ", ")
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// handleKey routes keyboard input by phase.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.phase == phaseRunning {
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
	case " ":
		if len(m.rows) > 0 {
			m.rows[m.cursor].selected = !m.rows[m.cursor].selected
		}
	case "a":
		for i := range m.rows {
			m.rows[i].selected = true
		}
	case "n":
		for i := range m.rows {
			m.rows[i].selected = false
		}
	case "enter":
		if m.phase == phaseDone {
			return m, popCmd()
		}
		m.phase = phaseRunning
		return m, runInstall(m.targetDir, m.selectedProfiles())
	case "esc", "left", "q":
		return m, popCmd()
	}
	return m, nil
}

// selectedProfiles returns the profiles the user checked.
func (m Model) selectedProfiles() []localprofile.Profile {
	out := make([]localprofile.Profile, 0, len(m.rows))
	for _, r := range m.rows {
		if r.selected {
			out = append(out, r.profile)
		}
	}
	return out
}

// SetSize stores the terminal dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// View renders the screen inside a FrameCompact border.
func (m Model) View() string {
	innerWidth := styles.CompactInnerWidth
	header := banner.RenderCompact(innerWidth, "local · install")

	var body strings.Builder
	body.WriteString(styles.Help.Render("target: "+m.targetDir) + "\n\n")
	body.WriteString(styles.SectionTitle.Render("Pick profiles to install:"))
	body.WriteString("\n\n")

	for i, r := range m.rows {
		var checkbox string
		if r.selected {
			checkbox = styles.CheckOn.Render("[x]")
		} else {
			checkbox = styles.CheckOff.Render("[ ]")
		}
		cursor := " "
		if i == m.cursor && m.phase == phaseIdle {
			cursor = styles.Arrow.Render("▸")
		}
		name := r.profile.DisplayName
		if r.installed {
			name += " " + styles.Dim("(installed)")
		}
		nameStyled := styles.MenuItem.Render(name)
		if r.selected {
			nameStyled = styles.MenuItemSelected.Render(name)
		}
		body.WriteString(fmt.Sprintf("%s%s %s\n", cursor, checkbox, nameStyled))
		body.WriteString("   " + styles.Help.Render(r.profile.Description) + "\n")
	}

	if m.phase == phaseRunning {
		body.WriteString("\n" + styles.WarningText.Render("Installing...") + "\n")
	}

	if m.toast != "" {
		body.WriteString("\n")
		if m.toastErr {
			body.WriteString(styles.ProgressError.Render(m.toast))
		} else {
			body.WriteString(styles.Toast.Render(m.toast))
		}
	}

	return styles.FrameCompact(header, body.String(), m.Footer(), styles.CompactWidth)
}

// Footer returns the key-hint line for the screen.
func (m Model) Footer() string {
	switch m.phase {
	case phaseRunning:
		return styles.FooterHints("please wait...", "")
	case phaseDone:
		return styles.FooterHints("enter/esc", "back")
	}
	return styles.FooterHints("↑/↓", "move", "space", "toggle", "enter", "install", "esc", "back")
}

// runInstall applies every selected profile to targetDir.
func runInstall(targetDir string, profiles []localprofile.Profile) tea.Cmd {
	return func() tea.Msg {
		var successes []string
		for _, p := range profiles {
			if err := p.Install(targetDir); err != nil {
				return doneMsg{successes: successes, err: fmt.Errorf("%s: %w", p.Name, err)}
			}
			successes = append(successes, p.DisplayName)
		}
		return doneMsg{successes: successes}
	}
}

// popCmd is the canonical emitter for "return to caller".
func popCmd() tea.Cmd {
	return func() tea.Msg { return messages.PopScreenMsg{} }
}
