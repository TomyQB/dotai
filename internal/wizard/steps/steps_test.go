package steps_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/provider"
	"github.com/TomyQB/dotai/internal/wizard"
	"github.com/TomyQB/dotai/internal/wizard/steps"
)

// newTestProvider creates a TestProvider rooted at t.TempDir().
func newTestProvider(t *testing.T) *provider.TestProvider {
	t.Helper()
	return provider.NewTestProvider(t.TempDir())
}

// pressKey sends a tea.KeyMsg to a tea.Model and returns the updated model.
func pressKey(m tea.Model, key string) tea.Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return updated
}

// pressSpecialKey sends a tea.KeyMsg with a special key type.
func pressSpecialKey(m tea.Model, keyType tea.KeyType) tea.Model {
	updated, _ := m.Update(tea.KeyMsg{Type: keyType})
	return updated
}

// ── Profile ──────────────────────────────────────────────────────────────────

func TestProfileName(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewProfile(prov)
	if s.Name() != "Profile" {
		t.Errorf("Name() = %q, want %q", s.Name(), "Profile")
	}
}

func TestProfileNeverSkipped(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewProfile(prov)
	// Profile is mandatory — Skipped() must always return false.
	if s.Skipped() {
		t.Error("Skipped() = true, want false (profile is mandatory)")
	}
}

func TestProfileApplyPersonal(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewProfile(prov)
	s.Init()

	// Default cursor is profilePersonal (0); pressing enter confirms it.
	updated := pressSpecialKey(s, tea.KeyEnter)
	confirmed := updated.(*steps.ProfileModel)

	state := &wizard.WizardState{}
	confirmed.Apply(state)

	if state.ProfileType != wizard.ProfilePersonal {
		t.Errorf("ProfileType = %v, want ProfilePersonal", state.ProfileType)
	}
}

func TestProfileApplyWork(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewProfile(prov)
	s.Init()

	// Move cursor down once to select work profile, then confirm.
	afterDown := pressKey(s, "j")
	confirmed := pressSpecialKey(afterDown, tea.KeyEnter).(*steps.ProfileModel)

	state := &wizard.WizardState{}
	confirmed.Apply(state)

	if state.ProfileType != wizard.ProfileWork {
		t.Errorf("ProfileType = %v, want ProfileWork", state.ProfileType)
	}
}

func TestProfileApplyEnablesSettings(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewProfile(prov)
	s.Init()

	updated := pressSpecialKey(s, tea.KeyEnter)
	confirmed := updated.(*steps.ProfileModel)

	state := &wizard.WizardState{}
	confirmed.Apply(state)

	if !state.SettingsEnabled {
		t.Error("SettingsEnabled = false after profile Apply, want true (always enabled)")
	}
	if state.Patch.BaseSettings == nil {
		t.Error("Patch.BaseSettings is nil after profile Apply, want populated map")
	}
}

func TestProfileApplyEnablesStatusline(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewProfile(prov)
	s.Init()

	updated := pressSpecialKey(s, tea.KeyEnter)
	confirmed := updated.(*steps.ProfileModel)

	state := &wizard.WizardState{}
	confirmed.Apply(state)

	if !state.StatuslineEnabled {
		t.Error("StatuslineEnabled = false after profile Apply, want true (always enabled)")
	}
	if !strings.Contains(state.Patch.StatusLine.Command, "statusline-command.sh") {
		t.Errorf("StatusLine.Command = %q, want it to contain statusline-command.sh",
			state.Patch.StatusLine.Command)
	}
}

func TestProfileApplyEnablesGitBranchHook(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewProfile(prov)
	s.Init()

	updated := pressSpecialKey(s, tea.KeyEnter)
	confirmed := updated.(*steps.ProfileModel)

	state := &wizard.WizardState{}
	confirmed.Apply(state)

	if !state.GitBranchHookEnabled {
		t.Error("GitBranchHookEnabled = false after profile Apply, want true (always enabled)")
	}
	if len(state.Patch.Hooks) == 0 {
		t.Fatal("Patch.Hooks is empty after profile Apply, want at least one hook entry")
	}
	hook := state.Patch.Hooks[0]
	if hook.Event != "PreToolUse" {
		t.Errorf("hook.Event = %q, want PreToolUse", hook.Event)
	}
	if !strings.Contains(hook.Command, "git-branch-check.sh") {
		t.Errorf("hook.Command = %q, want it to contain git-branch-check.sh", hook.Command)
	}
}

func TestProfileSummaryLinePersonal(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewProfile(prov)
	s.Init()

	// Default cursor is personal; confirm it.
	updated := pressSpecialKey(s, tea.KeyEnter)
	confirmed := updated.(*steps.ProfileModel)

	got := confirmed.SummaryLine()
	if got != "personal — full config" {
		t.Errorf("SummaryLine() = %q, want %q", got, "personal — full config")
	}
}

func TestProfileSummaryLineWork(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewProfile(prov)
	s.Init()

	afterDown := pressKey(s, "j")
	confirmed := pressSpecialKey(afterDown, tea.KeyEnter).(*steps.ProfileModel)

	got := confirmed.SummaryLine()
	if got != "work — full config" {
		t.Errorf("SummaryLine() = %q, want %q", got, "work — full config")
	}
}

// CursorCannotExceedWork verifies that pressing down past Work does not overflow.
func TestProfileCursorBounded(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewProfile(prov)
	s.Init()

	// Press down many times — cursor must stay at Work (1).
	m := tea.Model(s)
	for i := 0; i < 5; i++ {
		m = pressKey(m, "j")
	}
	confirmed := pressSpecialKey(m, tea.KeyEnter).(*steps.ProfileModel)

	state := &wizard.WizardState{}
	confirmed.Apply(state)

	if state.ProfileType != wizard.ProfileWork {
		t.Errorf("ProfileType = %v after excessive down presses, want ProfileWork", state.ProfileType)
	}
}

// ── Skills ───────────────────────────────────────────────────────────────────

func TestSkillsName(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewSkills(prov)
	if s.Name() != "Global Skills" {
		t.Errorf("Name() = %q, want %q", s.Name(), "Global Skills")
	}
}

func TestSkillsApplyAllSelectedByDefault(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewSkills(prov)
	s.Init()

	// Press enter to confirm without deselecting anything.
	updated := pressSpecialKey(s, tea.KeyEnter)
	confirmed := updated.(*steps.SkillsModel)

	state := &wizard.WizardState{}
	confirmed.Apply(state)

	// By default all skills are selected; there should be at least one.
	if len(state.SelectedSkills) == 0 {
		t.Error("SelectedSkills is empty after confirming with all defaults, want at least one")
	}
}

func TestSkillsApplyNoneSelected(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewSkills(prov)
	s.Init()

	// Press 'n' to deselect all and confirm immediately.
	updated := pressKey(s, "n")
	confirmed := updated.(*steps.SkillsModel)

	state := &wizard.WizardState{}
	confirmed.Apply(state)

	if len(state.SelectedSkills) != 0 {
		t.Errorf("SelectedSkills = %v, want empty after deselecting all", state.SelectedSkills)
	}
}

func TestSkillsSkippedWhenNoneSelected(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewSkills(prov)
	s.Init()

	updated := pressKey(s, "n")
	confirmed := updated.(*steps.SkillsModel)

	if !confirmed.Skipped() {
		t.Error("Skipped() = false after deselecting all, want true")
	}
}

// ── Memcli ───────────────────────────────────────────────────────────────────

func TestMemcliName(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewMemcli(prov)
	if s.Name() != "Memcli" {
		t.Errorf("Name() = %q, want %q", s.Name(), "Memcli")
	}
}

func TestMemcliSkippedByDefault(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewMemcli(prov)
	if !s.Skipped() {
		t.Error("Skipped() = false before confirmation, want true")
	}
}

func TestMemcliApplyWhenConfirmedYes(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewMemcli(prov)

	// Default cursor is memcliYes (0); pressing enter confirms it.
	updated := pressSpecialKey(s, tea.KeyEnter)
	confirmed := updated.(*steps.MemcliModel)

	state := &wizard.WizardState{}
	confirmed.Apply(state)

	if !state.MemcliEnabled {
		t.Error("MemcliEnabled = false after confirming Yes, want true")
	}
	if len(state.Patch.Hooks) == 0 {
		t.Fatal("Patch.Hooks is empty after memcli Apply, want Stop hook")
	}
	hook := state.Patch.Hooks[0]
	if hook.Event != "Stop" {
		t.Errorf("hook.Event = %q, want Stop", hook.Event)
	}
	if !strings.Contains(hook.Command, "stop-hook.sh") {
		t.Errorf("hook.Command = %q, want it to contain stop-hook.sh", hook.Command)
	}
}

func TestMemcliApplyWhenConfirmedNo(t *testing.T) {
	prov := newTestProvider(t)
	s := steps.NewMemcli(prov)

	// Move cursor down to No, then confirm.
	afterDown := pressKey(s, "j")
	updated := pressSpecialKey(afterDown, tea.KeyEnter)
	confirmed := updated.(*steps.MemcliModel)

	state := &wizard.WizardState{}
	confirmed.Apply(state)

	if state.MemcliEnabled {
		t.Error("MemcliEnabled = true after confirming No, want false")
	}
	if len(state.Patch.Hooks) != 0 {
		t.Error("Patch.Hooks non-empty after confirming No, want empty")
	}
}
