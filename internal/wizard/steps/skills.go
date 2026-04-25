package steps

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TomyQB/dotai/internal/provider"
	"github.com/TomyQB/dotai/internal/tui/styles"
	"github.com/TomyQB/dotai/internal/wizard"
)

// SkillInfo holds the metadata and selection state for a single installable skill.
type SkillInfo struct {
	DirName     string
	DisplayName string
	Description string
	Selected    bool
	Installed   bool
}

// SkillsModel is the wizard step that lets the user choose which global skills
// to install.
type SkillsModel struct {
	prov        provider.Provider
	skills      []SkillInfo
	cursor      int
	autoSkipped bool
	confirmed   bool
	width       int
	height      int
}

// NewSkills constructs a SkillsModel for the given provider.
func NewSkills(prov provider.Provider) *SkillsModel {
	return &SkillsModel{prov: prov}
}

// availableSkills returns the canonical list of skills in display order.
func availableSkills() []SkillInfo {
	return []SkillInfo{
		{DirName: "readme-generator", DisplayName: "Readme Generator", Description: "Generate professional READMEs"},
		{DirName: "skill-creator", DisplayName: "Skill Creator", Description: "Create new AI agent skills"},
		{DirName: "commit-and-push", DisplayName: "Commit & Push", Description: "Conventional commits workflow"},
		{DirName: "create-pr", DisplayName: "Create PR", Description: "Open/refresh GitHub PRs with functional context"},
		{DirName: "full-review", DisplayName: "Full Review", Description: "Audit + review + simplify quality gate"},
		{DirName: "junit-mockito", DisplayName: "JUnit Mockito", Description: "Java unit testing patterns"},
		{DirName: "owasp-audit", DisplayName: "OWASP Audit", Description: "OWASP ASVS security audit"},
		{DirName: "web3-audit", DisplayName: "Web3 Audit", Description: "Web3 security audit"},
		{DirName: "web3-review", DisplayName: "Web3 Review", Description: "Web3 code review"},
		{DirName: "java-review", DisplayName: "Java Review", Description: "Java code review"},
		{DirName: "java-spring-boot", DisplayName: "Spring Boot", Description: "Spring Boot patterns"},
	}
}

// Init builds the skill list and checks which are already installed.
// If the provider does not support skills, the step auto-skips.
func (m *SkillsModel) Init() tea.Cmd {
	if !m.prov.SupportsFeature(provider.FeatureSkills) {
		m.autoSkipped = true
		return func() tea.Msg { return wizard.StepAutoSkipMsg{} }
	}

	skills := availableSkills()

	configDir, err := m.prov.ConfigDir()
	if err != nil {
		configDir = ""
	}
	skillsRelDir := m.prov.SkillsDir()

	for i := range skills {
		skills[i].Selected = true // all selected by default
		if configDir != "" {
			skillPath := filepath.Join(configDir, skillsRelDir, skills[i].DirName)
			if _, statErr := os.Stat(skillPath); statErr == nil {
				skills[i].Installed = true
			}
		}
	}

	m.skills = skills
	return nil
}

// Update handles keyboard input for the skills step.
func (m *SkillsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.skills)-1 {
				m.cursor++
			}
		case " ":
			if len(m.skills) > 0 {
				m.skills[m.cursor].Selected = !m.skills[m.cursor].Selected
			}
		case "a":
			for i := range m.skills {
				m.skills[i].Selected = true
			}
		case "n":
			for i := range m.skills {
				m.skills[i].Selected = false
			}
			m.confirmed = true
			return m, func() tea.Msg { return wizard.StepCompleteMsg{} }
		case "enter", "y", "right":
			m.confirmed = true
			return m, func() tea.Msg { return wizard.StepCompleteMsg{} }
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View renders the skill checklist.
func (m *SkillsModel) View() string {
	var b strings.Builder

	b.WriteString(styles.SectionTitle.Render("Select skills to install:"))
	b.WriteString("\n\n")

	for i, skill := range m.skills {
		// Checkbox
		var checkbox string
		if skill.Selected {
			checkbox = styles.CheckOn.Render("[x]")
		} else {
			checkbox = styles.CheckOff.Render("[ ]")
		}

		// Cursor arrow
		var cursor string
		if i == m.cursor {
			cursor = styles.Arrow.Render("▸")
		} else {
			cursor = " "
		}

		// Display name — bold+primary when selected, dim otherwise
		var nameStyle lipgloss.Style
		if skill.Selected {
			nameStyle = styles.MenuItemSelected
		} else {
			nameStyle = styles.MenuItem
		}
		name := nameStyle.Width(18).Render(skill.DisplayName)

		// Description with optional (installed) suffix
		desc := skill.Description
		if skill.Installed {
			desc += " " + styles.Dim("(installed)")
		}
		descStr := styles.Help.Render(desc)

		line := fmt.Sprintf("%s%s %s  %s", cursor, checkbox, name, descStr)
		b.WriteString(line + "\n")
	}

	return b.String()
}

// Apply writes the selected skill directory names into the wizard state.
func (m *SkillsModel) Apply(state *wizard.WizardState) {
	var selected []string
	for _, s := range m.skills {
		if s.Selected {
			selected = append(selected, s.DirName)
		}
	}
	state.SelectedSkills = selected
}

// SummaryLine returns a human-readable one-liner for the summary screen.
func (m *SkillsModel) SummaryLine() string {
	if m.autoSkipped {
		return "none"
	}
	count := 0
	for _, s := range m.skills {
		if s.Selected {
			count++
		}
	}
	if count == 0 {
		return "none"
	}
	return fmt.Sprintf("%d of %d selected", count, len(m.skills))
}

// Skipped reports whether the step was effectively skipped.
func (m *SkillsModel) Skipped() bool {
	if m.autoSkipped {
		return true
	}
	count := 0
	for _, s := range m.skills {
		if s.Selected {
			count++
		}
	}
	return count == 0
}

// Name returns the step identifier.
func (m *SkillsModel) Name() string { return "Global Skills" }

// Description returns a short explanation shown in the step header.
func (m *SkillsModel) Description() string { return "AI coding skills to install globally" }

// Footer returns key-binding hints for this step.
func (m *SkillsModel) Footer() string {
	return styles.Footer.Render("↑/↓ move • space toggle • a all • n none • enter/→ confirm • ← back • esc menu")
}

// SetSize notifies the step of the current terminal dimensions.
func (m *SkillsModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Ensure SkillsModel satisfies the Step interface at compile time.
var _ wizard.Step = (*SkillsModel)(nil)
