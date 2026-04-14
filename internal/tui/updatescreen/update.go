// Package updatescreen implements the Update screen for dotai.
// It detects the components the user already has installed, shows them
// read-only, and re-applies the embedded templates to each on confirmation.
// Components that are not installed are left untouched.
package updatescreen

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/detect"
	"github.com/TomyQB/dotai/internal/provider"
	"github.com/TomyQB/dotai/internal/tui/banner"
	"github.com/TomyQB/dotai/internal/tui/messages"
	"github.com/TomyQB/dotai/internal/tui/styles"
	"github.com/TomyQB/dotai/internal/wizard"
)

// planMsg delivers the detection snapshot and the pre-computed plan preview
// lines back to the model after the async Init pass.
type planMsg struct {
	preview []previewRow
}

// previewRow is one line of the "will update" list shown before confirmation.
type previewRow struct {
	label     string
	installed bool
	detail    string
}

// updateDoneMsg signals the Update operation has finished.
type updateDoneMsg struct {
	report wizard.UpdateReport
	err    error
}

// Model is the Bubble Tea model for the update screen.
type Model struct {
	prov          provider.Provider
	preview       []previewRow
	loaded        bool
	busy          bool
	done          bool
	report        wizard.UpdateReport
	toast         string
	toastErr      bool
	cursor        int
	width, height int
}

// New returns an initialised update Model for the given provider.
func New(prov provider.Provider) Model {
	return Model{prov: prov}
}

// Title satisfies messages.Screen.
func (m Model) Title() string { return "update" }

// Init triggers an async detection pass that feeds the preview list.
func (m Model) Init() tea.Cmd {
	prov := m.prov
	return func() tea.Msg {
		return planMsg{preview: buildPreview(prov)}
	}
}

// Update handles messages and key events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case planMsg:
		m.preview = msg.preview
		m.loaded = true
		return m, nil

	case updateDoneMsg:
		m.busy = false
		m.done = true
		m.report = msg.report
		if msg.err != nil {
			m.toast = "update failed: " + msg.err.Error()
			m.toastErr = true
		} else {
			m.toast = "update completed"
			m.toastErr = false
		}
		return m, nil

	case tea.KeyMsg:
		if m.busy {
			return m, nil
		}
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < 1 {
				m.cursor++
			}
		case "enter":
			// Two actions: 0=Run Update, 1=Back.
			switch m.cursor {
			case 0:
				if m.done || !anyInstalled(m.preview) {
					return m, func() tea.Msg { return messages.PopScreenMsg{} }
				}
				m.busy = true
				m.toast = ""
				return m, runUpdateCmd(m.prov)
			case 1:
				return m, func() tea.Msg { return messages.PopScreenMsg{} }
			}
		case "esc", "q":
			return m, func() tea.Msg { return messages.PopScreenMsg{} }
		}
	}
	return m, nil
}

// SetSize stores the terminal dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// View renders the update screen inside a FrameCompact border.
func (m Model) View() string {
	innerWidth := styles.CompactInnerWidth

	header := banner.RenderCompact(innerWidth, "update")

	var body strings.Builder

	if !m.loaded {
		body.WriteString(styles.Help.Render("Detecting installed components..."))
		return styles.FrameCompact(header, body.String(), m.Footer(), styles.CompactWidth)
	}

	if m.done {
		body.WriteString(renderReport(m.report))
		body.WriteString("\n")
		body.WriteString(renderActions(m.cursor, true))
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

	// Plan preview.
	body.WriteString(styles.SectionTitle.Render("Will re-apply to installed components:"))
	body.WriteString("\n\n")
	anyInst := false
	for _, row := range m.preview {
		if row.installed {
			anyInst = true
			body.WriteString(styles.CheckOn.Render("✓") + " " +
				styles.MenuItem.Render(row.label))
			if row.detail != "" {
				body.WriteString("  " + styles.Dim(row.detail))
			}
			body.WriteString("\n")
		} else {
			body.WriteString(styles.CheckOff.Render("✗") + " " +
				styles.Help.Render(row.label+" (not installed — skipped)"))
			body.WriteString("\n")
		}
	}
	if !anyInst {
		body.WriteString("\n")
		body.WriteString(styles.WarningText.Render("Nothing to update — run Install first."))
	}

	body.WriteString("\n")
	body.WriteString(renderActions(m.cursor, false))

	if m.busy {
		body.WriteString("\n")
		body.WriteString(styles.WarningText.Render("Applying..."))
	}

	return styles.FrameCompact(header, body.String(), m.Footer(), styles.CompactWidth)
}

// Footer returns the key-hint line for this screen.
func (m Model) Footer() string {
	return styles.FooterHints("↑/↓", "navigate", "enter", "select", "esc", "back")
}

// renderActions renders the two bottom buttons (Run Update / Back).
// When done is true, the primary button becomes "Back" alone — the update
// can only be run once per session.
func renderActions(cursor int, done bool) string {
	var b strings.Builder
	primary := "Run Update"
	if done {
		primary = "Done"
	}
	labels := []string{primary, "Back"}
	for i, label := range labels {
		if i == cursor {
			b.WriteString(styles.Arrow.Render("▸ ") + styles.MenuItemSelected.Render(label))
		} else {
			b.WriteString("  " + styles.MenuItem.Render(label))
		}
		if i < len(labels)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderReport renders the post-update summary: updated rows first, then any
// skipped rows with their reason.
func renderReport(r wizard.UpdateReport) string {
	var b strings.Builder
	b.WriteString(styles.SectionTitle.Render("Updated:"))
	b.WriteString("\n")
	if len(r.Updated) == 0 {
		b.WriteString(styles.Help.Render("  (nothing)"))
		b.WriteString("\n")
	}
	for _, item := range r.Updated {
		b.WriteString(styles.CheckOn.Render("  ✓ ") + styles.MenuItem.Render(item))
		b.WriteString("\n")
	}
	if len(r.Skipped) > 0 {
		b.WriteString("\n")
		b.WriteString(styles.SectionTitle.Render("Skipped:"))
		b.WriteString("\n")
		for _, s := range r.Skipped {
			b.WriteString(styles.CheckOff.Render("  ✗ ") +
				styles.Help.Render(s.Component+" — "+s.Reason))
			b.WriteString("\n")
		}
	}
	return b.String()
}

// buildPreview runs CheckAll and converts the result into preview rows in a
// stable display order. Shown to the user before confirmation.
func buildPreview(prov provider.Provider) []previewRow {
	results := detect.CheckAll(prov)
	byComponent := make(map[detect.Component]detect.CheckResult, len(results))
	for _, r := range results {
		byComponent[r.Component] = r
	}

	order := []struct {
		comp  detect.Component
		label string
	}{
		{detect.Profile, "CLAUDE.md"},
		{detect.Settings, "settings.json"},
		{detect.Statusline, "statusline"},
		{detect.GitHook, "git-branch-check hook"},
		{detect.Skills, "skills"},
		{detect.Memcli, "memcli"},
	}

	rows := make([]previewRow, 0, len(order))
	for _, o := range order {
		r := byComponent[o.comp]
		rows = append(rows, previewRow{
			label:     o.label,
			installed: r.Installed,
			detail:    r.Detail,
		})
	}
	return rows
}

// anyInstalled reports whether at least one row in the preview is installed.
func anyInstalled(rows []previewRow) bool {
	for _, r := range rows {
		if r.installed {
			return true
		}
	}
	return false
}

// runUpdateCmd invokes wizard.Update asynchronously and reports the outcome.
func runUpdateCmd(prov provider.Provider) tea.Cmd {
	return func() tea.Msg {
		rep, err := wizard.Update(prov)
		return updateDoneMsg{report: rep, err: err}
	}
}
