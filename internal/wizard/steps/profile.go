// Package steps contains the individual wizard step implementations.
package steps

import (
	"encoding/json"
	"io/fs"
	"strings"
	"text/template"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"

	"github.com/TomyQB/dotai/internal/assets"
	"github.com/TomyQB/dotai/internal/provider"
	"github.com/TomyQB/dotai/internal/tui/styles"
	"github.com/TomyQB/dotai/internal/wizard"
)

// profileOption is the index of a profile choice.
type profileOption int

const (
	profilePersonal profileOption = 0
	profileWork     profileOption = 1
)

// ProfileModel is the wizard step that chooses an environment (Personal or Work).
// It is mandatory — there is no Skip option. Confirming it automatically enables
// settings, statusline, git-branch-hook, and the matching CLAUDE.md profile.
type ProfileModel struct {
	prov       provider.Provider
	cursor     profileOption
	previewing bool
	viewport   viewport.Model
	vpReady    bool
	profiles   [2]string // preloaded content: [0]=personal, [1]=work
	confirmed  bool
	width      int
	height     int
}

// NewProfile constructs a ProfileModel for the given provider.
func NewProfile(prov provider.Provider) *ProfileModel {
	return &ProfileModel{prov: prov}
}

// Init loads the profile files and, if the provider does not support profiles,
// emits a StepAutoSkipMsg so the wizard skips this step entirely.
func (m *ProfileModel) Init() tea.Cmd {
	if !m.prov.SupportsFeature(provider.FeatureProfiles) {
		return func() tea.Msg { return wizard.StepAutoSkipMsg{} }
	}

	// Preload profile contents; silently leave empty on error — the step is
	// still operable, it just cannot show a preview.
	if raw, err := fs.ReadFile(assets.FS, "files/profiles/personal.md"); err == nil {
		m.profiles[0] = string(raw)
	}
	if raw, err := fs.ReadFile(assets.FS, "files/profiles/work.md"); err == nil {
		m.profiles[1] = string(raw)
	}

	return nil
}

// Update handles keyboard input for the profile step.
func (m *ProfileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.previewing {
		return m.updatePreview(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > profilePersonal {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < profileWork {
				m.cursor++
			}
		case " ", "p":
			m.enterPreview()
		case "enter":
			m.confirmed = true
			return m, func() tea.Msg { return wizard.StepCompleteMsg{} }
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// enterPreview initialises the viewport with the current profile's content.
func (m *ProfileModel) enterPreview() {
	content := m.profiles[m.cursor]
	if content == "" {
		return
	}

	// Reserve 6 lines for chrome (header + footer).
	vpHeight := m.height - 6
	if vpHeight < 3 {
		vpHeight = 3
	}
	vpWidth := m.width - 4
	if vpWidth < 20 {
		vpWidth = 20
	}

	m.viewport = viewport.New(vpWidth, vpHeight)
	m.viewport.SetContent(content)
	m.vpReady = true
	m.previewing = true
}

// updatePreview handles keyboard input while the viewport is open.
func (m *ProfileModel) updatePreview(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.previewing = false
			m.vpReady = false
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the step. When previewing, it shows the viewport.
func (m *ProfileModel) View() string {
	if m.previewing {
		return m.viewPreview()
	}
	return m.viewSelection()
}

func (m *ProfileModel) viewSelection() string {
	var b strings.Builder

	b.WriteString(styles.SectionTitle.Render("Select your environment:"))
	b.WriteString("\n\n")

	type row struct {
		opt  profileOption
		name string
		desc string
	}
	rows := []row{
		{profilePersonal, "Personal", "opinionated defaults for personal projects"},
		{profileWork, "Work", "conservative, team-friendly defaults"},
	}

	for _, r := range rows {
		var line string
		if m.cursor == r.opt {
			arrow := styles.Arrow.Render("▸")
			label := styles.MenuItemSelected.Render(r.name)
			desc := styles.Help.Render("— " + r.desc)
			line = arrow + " " + label + "  " + desc
		} else {
			label := styles.MenuItem.Render(r.name)
			desc := styles.Help.Render("— " + r.desc)
			line = "  " + label + "  " + desc
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.Help.Render("Installs: settings.json + statusline + git-branch hook + CLAUDE.md"))

	return b.String()
}

func (m *ProfileModel) viewPreview() string {
	var b strings.Builder

	title := "Personal"
	if m.cursor == profileWork {
		title = "Work"
	}
	b.WriteString(styles.SectionTitle.Render("Preview: " + title))
	b.WriteString("\n\n")
	b.WriteString(m.viewport.View())
	b.WriteString("\n\n")
	b.WriteString(styles.Footer.Render("esc back • ↑/↓ scroll"))
	return b.String()
}

// Apply writes the selected profile type into the wizard state and enables all
// always-on features: settings, statusline, and git-branch-hook.
func (m *ProfileModel) Apply(state *wizard.WizardState) {
	if m.cursor == profilePersonal {
		state.ProfileType = wizard.ProfilePersonal
	} else {
		state.ProfileType = wizard.ProfileWork
	}

	configDir, err := m.prov.ConfigDir()
	if err != nil {
		return
	}

	// Settings — always installed.
	state.SettingsEnabled = true
	if raw, err := fs.ReadFile(assets.FS, "files/settings/settings.json.tmpl"); err == nil {
		if tmpl, err := template.New("settings").Parse(string(raw)); err == nil {
			data := struct{ ClaudeDir string }{ClaudeDir: configDir}
			var sb strings.Builder
			if err := tmpl.Execute(&sb, data); err == nil {
				var base map[string]any
				if err := json.Unmarshal([]byte(sb.String()), &base); err == nil {
					state.Patch.BaseSettings = base
				}
			}
		}
	}

	// Statusline — always installed.
	state.StatuslineEnabled = true
	state.Patch.StatusLine = wizard.StatusLineConfig{
		Command: "bash " + configDir + "/statusline-command.sh",
	}

	// Git branch check hook — always installed.
	state.GitBranchHookEnabled = true
	state.Patch.Hooks = append(state.Patch.Hooks, wizard.HookEntry{
		Event:   "PreToolUse",
		Matcher: "Edit|Write|Agent",
		Command: "bash " + configDir + "/" + m.prov.ToolDir() + "/hooks/git-branch-check.sh",
	})
}

// SummaryLine returns a human-readable one-liner for the summary screen.
func (m *ProfileModel) SummaryLine() string {
	if m.cursor == profilePersonal {
		return "personal — full config"
	}
	return "work — full config"
}

// Skipped always returns false — environment selection is mandatory.
func (m *ProfileModel) Skipped() bool { return false }

// Name returns the step identifier.
func (m *ProfileModel) Name() string { return "Profile" }

// Description returns a short explanation shown in the step header.
func (m *ProfileModel) Description() string {
	return "Choose your environment — installs all config + CLAUDE.md"
}

// Footer returns context-sensitive key-binding hints.
func (m *ProfileModel) Footer() string {
	if m.previewing {
		return styles.Footer.Render("esc back • ↑/↓ scroll")
	}
	return styles.Footer.Render("↑/↓ select • space preview • enter confirm")
}

// SetSize notifies the step of the current terminal dimensions.
func (m *ProfileModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	if m.vpReady {
		vpHeight := height - 6
		if vpHeight < 3 {
			vpHeight = 3
		}
		vpWidth := width - 4
		if vpWidth < 20 {
			vpWidth = 20
		}
		m.viewport.Width = vpWidth
		m.viewport.Height = vpHeight
	}
}

// Ensure ProfileModel satisfies the Step interface at compile time.
var _ wizard.Step = (*ProfileModel)(nil)
