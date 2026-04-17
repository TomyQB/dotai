// Package localstatus implements the Status screen for `dotai local`.
// It reads targetDir/.claude/ and reports, per profile, which skills and
// CLAUDE.md artifacts are present.
package localstatus

import (
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/localprofile"
	"github.com/TomyQB/dotai/internal/tui/banner"
	"github.com/TomyQB/dotai/internal/tui/messages"
	"github.com/TomyQB/dotai/internal/tui/styles"
)

// profileReport is the computed status of a single profile in the target dir.
type profileReport struct {
	profile        localprofile.Profile
	presentSkills  []string
	missingSkills  []string
	claudeMdExists bool
}

// reportMsg delivers the computed reports back to the model.
type reportMsg struct {
	reports    []profileReport
	claudeRoot bool
}

// Model is the Bubble Tea model for the local Status screen.
type Model struct {
	targetDir     string
	reports       []profileReport
	claudeRoot    bool
	loaded        bool
	width, height int
}

// New returns a Status screen bound to targetDir.
func New(targetDir string) Model {
	return Model{targetDir: targetDir}
}

// Title satisfies messages.Screen.
func (m Model) Title() string { return "local status" }

// Init kicks off an async scan of the target directory.
func (m Model) Init() tea.Cmd {
	target := m.targetDir
	return func() tea.Msg {
		profiles := localprofile.All()
		reports := make([]profileReport, len(profiles))
		for i, p := range profiles {
			var present, missing []string
			for _, s := range p.Skills {
				if localprofile.IsSkillInstalled(target, s) {
					present = append(present, s)
				} else {
					missing = append(missing, s)
				}
			}
			claudeMd := false
			if p.ClaudeMdAsset != "" {
				if _, err := os.Stat(localprofile.ClaudeMdPath(target)); err == nil {
					claudeMd = true
				}
			}
			reports[i] = profileReport{
				profile:        p,
				presentSkills:  present,
				missingSkills:  missing,
				claudeMdExists: claudeMd,
			}
		}
		_, rootErr := os.Stat(target + "/.claude")
		return reportMsg{reports: reports, claudeRoot: rootErr == nil}
	}
}

// Update handles incoming messages and key events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case reportMsg:
		m.reports = msg.reports
		m.claudeRoot = msg.claudeRoot
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
	header := banner.RenderCompact(innerWidth, "local · status")

	if !m.loaded {
		return styles.FrameCompact(header, "Scanning...", m.Footer(), styles.CompactWidth)
	}

	var body strings.Builder
	body.WriteString(styles.Help.Render("target: "+m.targetDir) + "\n")

	rootIcon := styles.CheckOff.Render("✗")
	rootText := styles.Help.Render(".claude/ does not exist")
	if m.claudeRoot {
		rootIcon = styles.CheckOn.Render("✓")
		rootText = styles.MenuItem.Render(".claude/ present")
	}
	body.WriteString(rootIcon + " " + rootText + "\n\n")

	for i, r := range m.reports {
		title := styles.SectionTitle.Render(r.profile.DisplayName)
		body.WriteString(title + "\n")

		total := len(r.profile.Skills)
		have := len(r.presentSkills)
		icon := styles.CheckOff.Render("✗")
		if have == total && total > 0 {
			icon = styles.CheckOn.Render("✓")
		} else if have > 0 {
			icon = styles.WarningText.Render("◐")
		}
		body.WriteString("  " + icon + " " +
			styles.MenuItem.Render(skillsLabel(have, total)) + "\n")

		for _, s := range r.profile.Skills {
			mark := styles.CheckOff.Render("  ✗ ")
			if contains(r.presentSkills, s) {
				mark = styles.CheckOn.Render("  ✓ ")
			}
			body.WriteString(mark + styles.Dim(s) + "\n")
		}

		if r.profile.ClaudeMdAsset != "" {
			icon := styles.CheckOff.Render("✗")
			text := "CLAUDE.md not present"
			if r.claudeMdExists {
				icon = styles.CheckOn.Render("✓")
				text = "CLAUDE.md present"
			}
			body.WriteString("  " + icon + " " + styles.MenuItem.Render(text) + "\n")
		}

		if i < len(m.reports)-1 {
			body.WriteString("\n")
		}
	}

	return styles.FrameCompact(header, body.String(), m.Footer(), styles.CompactWidth)
}

// Footer returns the key-hint line for the screen.
func (m Model) Footer() string {
	return styles.FooterHints("esc", "back")
}

// skillsLabel renders "3/5 skills present" or "0/5 skills present".
func skillsLabel(have, total int) string {
	return strconv.Itoa(have) + "/" + strconv.Itoa(total) + " skills present"
}

// contains reports whether haystack has needle.
func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
