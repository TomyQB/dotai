// Package localdoctor implements the Doctor diagnostic screen for `dotai local`.
// It performs lightweight checks on the target directory's .claude/ folder
// (exists, writable, skills subtree readable) and reports each with a badge.
package localdoctor

import (
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/localprofile"
	"github.com/TomyQB/dotai/internal/tui/banner"
	"github.com/TomyQB/dotai/internal/tui/messages"
	"github.com/TomyQB/dotai/internal/tui/styles"
)

// check is a single diagnostic row.
type check struct {
	label  string
	ok     bool
	detail string
}

// diagMsg carries the completed diagnostic rows back to the model.
type diagMsg []check

// Model is the Bubble Tea model for the local Doctor screen.
type Model struct {
	targetDir     string
	checks        []check
	loaded        bool
	width, height int
}

// New returns a Doctor screen bound to targetDir.
func New(targetDir string) Model {
	return Model{targetDir: targetDir}
}

// Title satisfies messages.Screen.
func (m Model) Title() string { return "local doctor" }

// Init runs the async diagnostic sweep.
func (m Model) Init() tea.Cmd {
	target := m.targetDir
	return func() tea.Msg {
		return diagMsg(runChecks(target))
	}
}

// Update handles incoming messages and key events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case diagMsg:
		m.checks = []check(msg)
		m.loaded = true
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "left":
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

// View renders the screen inside a FrameCompact border.
func (m Model) View() string {
	innerWidth := styles.CompactInnerWidth
	header := banner.RenderCompact(innerWidth, "local · doctor")

	if !m.loaded {
		return styles.FrameCompact(header, "Running diagnostics...", m.Footer(), styles.CompactWidth)
	}

	body := styles.Help.Render("target: "+m.targetDir) + "\n\n"
	for _, c := range m.checks {
		body += renderRow(c) + "\n"
	}

	return styles.FrameCompact(header, body, m.Footer(), styles.CompactWidth)
}

// Footer returns the key-hint line for the screen.
func (m Model) Footer() string {
	return styles.FooterHints("esc", "back")
}

// renderRow formats a single diagnostic row with badge, label, and detail.
func renderRow(c check) string {
	badge := styles.BadgeMissing()
	if c.ok {
		badge = styles.BadgeOK()
	}
	return styles.MenuItem.Render(c.label) +
		"  " + badge +
		"  " + styles.Dim(c.detail)
}

// runChecks computes every diagnostic row for targetDir.
func runChecks(targetDir string) []check {
	checks := []check{}

	targetInfo, err := os.Stat(targetDir)
	checks = append(checks, check{
		label:  "Target directory",
		ok:     err == nil && targetInfo.IsDir(),
		detail: targetDir,
	})

	claudeDir := filepath.Join(targetDir, ".claude")
	claudeInfo, claudeErr := os.Stat(claudeDir)
	claudeOK := claudeErr == nil && claudeInfo.IsDir()
	claudeDetail := ".claude/ exists"
	if !claudeOK {
		claudeDetail = ".claude/ not present (will be created on install)"
	}
	checks = append(checks, check{label: ".claude directory", ok: claudeOK, detail: claudeDetail})

	writable := isWritable(targetDir)
	writableDetail := "target directory is writable"
	if !writable {
		writableDetail = "target directory is not writable"
	}
	checks = append(checks, check{label: "Writable", ok: writable, detail: writableDetail})

	skillsDir := localprofile.SkillsDir(targetDir)
	skillsInfo, skillsErr := os.Stat(skillsDir)
	skillsOK := skillsErr == nil && skillsInfo.IsDir()
	skillsDetail := ".claude/skills/ exists"
	if !skillsOK {
		skillsDetail = ".claude/skills/ not present (will be created on install)"
	}
	checks = append(checks, check{label: "Skills directory", ok: skillsOK, detail: skillsDetail})

	return checks
}

// isWritable probes dir by creating and removing a temp file.
// Returns false on any error (permission denied, not a directory, etc.).
func isWritable(dir string) bool {
	if _, err := os.Stat(dir); err != nil {
		return false
	}
	f, err := os.CreateTemp(dir, ".dotai-probe-*")
	if err != nil {
		return false
	}
	_ = f.Close()
	_ = os.Remove(f.Name())
	return true
}
