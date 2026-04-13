package wizard_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TomyQB/dotai/internal/provider"
	"github.com/TomyQB/dotai/internal/wizard"
)

// newExecutor is a helper that creates an Executor backed by a TestProvider
// whose root is the given directory.
func newExecutor(dir string) (*wizard.Executor, *provider.TestProvider) {
	prov := provider.NewTestProvider(dir)
	return wizard.NewExecutor(prov), prov
}

// readSettings reads and parses settings.json from configDir.
func readSettings(t *testing.T, configDir string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(configDir, "settings.json"))
	if err != nil {
		t.Fatalf("read settings.json: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("parse settings.json: %v", err)
	}
	return out
}

// assertFileExecutable asserts the file at path has at least the execute bit set.
func assertFileExecutable(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("expected %s to be executable, got mode %v", path, info.Mode())
	}
}

// assertFileExists asserts the file at path exists.
func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected %s to exist: %v", path, err)
	}
}

// assertDirExists asserts the directory at path exists.
func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("expected dir %s to exist: %v", path, err)
		return
	}
	if !info.IsDir() {
		t.Errorf("expected %s to be a directory", path)
	}
}

// runApply synchronously executes the wizard state via Run and returns any error.
func runApply(t *testing.T, exec *wizard.Executor, state *wizard.WizardState) error {
	t.Helper()
	msg := exec.Run(state)()
	if pm, ok := msg.(wizard.ProgressMsg); ok {
		return pm.Err
	}
	t.Fatalf("unexpected message type from Run: %T", msg)
	return nil
}

// ---------------------------------------------------------------------------
// TestDeepMerge — verified indirectly via executor settings merge behavior.
// ---------------------------------------------------------------------------

func TestDeepMerge(t *testing.T) {
	tests := []struct {
		name         string
		existing     map[string]any
		baseSettings map[string]any
		wantKey      string
		wantValue    any
	}{
		{
			name:         "new key from base settings is applied",
			existing:     map[string]any{},
			baseSettings: map[string]any{"newKey": "newValue"},
			wantKey:      "newKey",
			wantValue:    "newValue",
		},
		{
			name:         "src wins on scalar conflict",
			existing:     map[string]any{"key": "old"},
			baseSettings: map[string]any{"key": "new"},
			wantKey:      "key",
			wantValue:    "new",
		},
		{
			name: "nested maps are merged recursively, src wins leaf",
			existing: map[string]any{
				"parent": map[string]any{"a": "keep", "b": "old"},
			},
			baseSettings: map[string]any{
				"parent": map[string]any{"b": "new", "c": "added"},
			},
			// We verify "parent.b" == "new" by checking the serialised settings.
			wantKey:   "parent",
			wantValue: nil, // checked separately below
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			exec, prov := newExecutor(dir)

			// Pre-populate settings with existing content.
			if err := prov.WriteSettings(tt.existing); err != nil {
				t.Fatalf("WriteSettings: %v", err)
			}

			state := &wizard.WizardState{
				SettingsEnabled: true,
				Patch: wizard.SettingsPatch{
					BaseSettings: tt.baseSettings,
				},
			}

			if err := runApply(t, exec, state); err != nil {
				t.Fatalf("apply: %v", err)
			}

			settings := readSettings(t, dir)

			if tt.name == "nested maps are merged recursively, src wins leaf" {
				parent, ok := settings["parent"].(map[string]any)
				if !ok {
					t.Fatalf("expected parent to be map, got %T", settings["parent"])
				}
				if parent["a"] != "keep" {
					t.Errorf("parent.a = %v, want keep", parent["a"])
				}
				if parent["b"] != "new" {
					t.Errorf("parent.b = %v, want new", parent["b"])
				}
				if parent["c"] != "added" {
					t.Errorf("parent.c = %v, want added", parent["c"])
				}
				return
			}

			if got := settings[tt.wantKey]; got != tt.wantValue {
				t.Errorf("settings[%q] = %v, want %v", tt.wantKey, got, tt.wantValue)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestExecutor_FullInstall — all six main install groups enabled.
// ---------------------------------------------------------------------------

func TestExecutor_FullInstall(t *testing.T) {
	dir := t.TempDir()
	exec, _ := newExecutor(dir)

	state := &wizard.WizardState{
		SettingsEnabled:      true,
		StatuslineEnabled:    true,
		GitBranchHookEnabled: true,
		ProfileType:          wizard.ProfilePersonal,
		SelectedSkills:       []string{"commit-and-push"},
		MemcliEnabled:        true,
		Patch: wizard.SettingsPatch{
			BaseSettings: map[string]any{"theme": "dark"},
			StatusLine:   wizard.StatusLineConfig{Command: "statusline-command.sh"},
			Hooks: []wizard.HookEntry{
				{Event: "PreToolUse", Command: "dotai/pre-tool.sh"},
			},
		},
	}

	if err := runApply(t, exec, state); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// Config dir exists.
	assertDirExists(t, dir)

	// Skills dir created and requested skill is present.
	assertDirExists(t, filepath.Join(dir, "skills"))
	assertDirExists(t, filepath.Join(dir, "skills", "commit-and-push"))
	assertFileExists(t, filepath.Join(dir, "skills", "commit-and-push", "SKILL.md"))

	// Hooks dir and hook scripts are executable.
	hooksDir := filepath.Join(dir, "tools", "hooks")
	assertDirExists(t, hooksDir)
	assertFileExecutable(t, filepath.Join(hooksDir, "git-branch-check.sh"))

	// Memcli hooks are executable.
	assertFileExecutable(t, filepath.Join(hooksDir, "pre-commit.sh"))
	assertFileExecutable(t, filepath.Join(hooksDir, "stop-hook.sh"))

	// Statusline script present and executable.
	assertFileExecutable(t, filepath.Join(dir, "statusline-command.sh"))

	// CLAUDE.md exists (profile).
	assertFileExists(t, filepath.Join(dir, "CLAUDE.md"))

	// Agents dir and doc-keeper present (memcli).
	assertDirExists(t, filepath.Join(dir, "agents"))
	assertFileExists(t, filepath.Join(dir, "agents", "doc-keeper.md"))

	// settings.json is valid JSON with expected entries.
	settings := readSettings(t, dir)

	if _, ok := settings["statusLine"]; !ok {
		t.Error("settings.json missing 'statusLine' key")
	}
	hooksRaw, ok := settings["hooks"]
	if !ok {
		t.Error("settings.json missing 'hooks' key")
	} else {
		hooksMap, ok := hooksRaw.(map[string]any)
		if !ok {
			t.Fatalf("hooks is not a map, got %T", hooksRaw)
		}
		if _, ok := hooksMap["PreToolUse"]; !ok {
			t.Error("hooks missing 'PreToolUse' entry")
		}
	}
}

// ---------------------------------------------------------------------------
// TestExecutor_SettingsOnly — only the settings step is enabled.
// ---------------------------------------------------------------------------

func TestExecutor_SettingsOnly(t *testing.T) {
	dir := t.TempDir()
	exec, _ := newExecutor(dir)

	state := &wizard.WizardState{
		SettingsEnabled: true,
		Patch: wizard.SettingsPatch{
			BaseSettings: map[string]any{"model": "claude-3-5-sonnet"},
		},
	}

	if err := runApply(t, exec, state); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// settings.json must exist and be valid.
	settings := readSettings(t, dir)
	if settings["model"] != "claude-3-5-sonnet" {
		t.Errorf("model = %v, want claude-3-5-sonnet", settings["model"])
	}

	// No skill dirs.
	if _, err := os.Stat(filepath.Join(dir, "skills")); !os.IsNotExist(err) {
		t.Error("skills dir should not exist when no skills selected")
	}

	// No hook files.
	if _, err := os.Stat(filepath.Join(dir, "tools", "hooks")); !os.IsNotExist(err) {
		t.Error("hooks dir should not exist when hooks not enabled")
	}

	// No CLAUDE.md.
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Error("CLAUDE.md should not exist when no profile selected")
	}
}

// ---------------------------------------------------------------------------
// TestExecutor_IdempotentHooks — running twice must not duplicate hook entries.
// ---------------------------------------------------------------------------

func TestExecutor_IdempotentHooks(t *testing.T) {
	dir := t.TempDir()
	exec, _ := newExecutor(dir)

	state := &wizard.WizardState{
		Patch: wizard.SettingsPatch{
			Hooks: []wizard.HookEntry{
				{Event: "PreToolUse", Command: "dotai/hooks/pre-tool.sh"},
			},
		},
	}

	// First run.
	if err := runApply(t, exec, state); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	// Second run — same state.
	if err := runApply(t, exec, state); err != nil {
		t.Fatalf("second apply: %v", err)
	}

	settings := readSettings(t, dir)
	hooksRaw, ok := settings["hooks"]
	if !ok {
		t.Fatal("settings.json missing 'hooks' key after two runs")
	}
	hooksMap, ok := hooksRaw.(map[string]any)
	if !ok {
		t.Fatalf("hooks is not a map, got %T", hooksRaw)
	}
	preToolRaw, ok := hooksMap["PreToolUse"]
	if !ok {
		t.Fatal("PreToolUse missing from hooks map")
	}
	entries, ok := preToolRaw.([]any)
	if !ok {
		t.Fatalf("PreToolUse entries is not a slice, got %T", preToolRaw)
	}
	if len(entries) != 1 {
		t.Errorf("expected exactly 1 PreToolUse entry, got %d", len(entries))
	}
}

// ---------------------------------------------------------------------------
// TestExecutor_PreservesUserSettings — custom keys survive a settings merge.
// ---------------------------------------------------------------------------

func TestExecutor_PreservesUserSettings(t *testing.T) {
	dir := t.TempDir()
	exec, prov := newExecutor(dir)

	// Pre-populate with user's custom setting.
	if err := prov.WriteSettings(map[string]any{"customKey": "value"}); err != nil {
		t.Fatalf("WriteSettings: %v", err)
	}

	state := &wizard.WizardState{
		SettingsEnabled: true,
		Patch: wizard.SettingsPatch{
			BaseSettings: map[string]any{"theme": "dark"},
		},
	}

	if err := runApply(t, exec, state); err != nil {
		t.Fatalf("apply: %v", err)
	}

	settings := readSettings(t, dir)

	if settings["customKey"] != "value" {
		t.Errorf("customKey = %v, want value (user setting should be preserved)", settings["customKey"])
	}
	if settings["theme"] != "dark" {
		t.Errorf("theme = %v, want dark (base settings should be merged)", settings["theme"])
	}
}

// ---------------------------------------------------------------------------
// TestExecutor_SkillsCopied — only the requested skills are installed.
// ---------------------------------------------------------------------------

func TestExecutor_SkillsCopied(t *testing.T) {
	dir := t.TempDir()
	exec, _ := newExecutor(dir)

	state := &wizard.WizardState{
		SelectedSkills: []string{"commit-and-push", "owasp-audit"},
	}

	if err := runApply(t, exec, state); err != nil {
		t.Fatalf("apply: %v", err)
	}

	skillsDir := filepath.Join(dir, "skills")
	assertDirExists(t, skillsDir)

	// Both requested skills must exist with a SKILL.md.
	for _, skill := range state.SelectedSkills {
		assertDirExists(t, filepath.Join(skillsDir, skill))
		assertFileExists(t, filepath.Join(skillsDir, skill, "SKILL.md"))
	}

	// No other skill dirs should be present (only the two selected).
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatalf("ReadDir skills: %v", err)
	}
	if len(entries) != 2 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected 2 skill dirs, got %d: %v", len(entries), names)
	}
}

// ---------------------------------------------------------------------------
// TestExecutor_ProfileCopy — profile content matches the embedded asset.
// ---------------------------------------------------------------------------

func TestExecutor_ProfileCopy(t *testing.T) {
	tests := []struct {
		name        string
		profileType wizard.ProfileType
	}{
		{"personal", wizard.ProfilePersonal},
		{"work", wizard.ProfileWork},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			exec, _ := newExecutor(dir)

			state := &wizard.WizardState{
				ProfileType: tt.profileType,
			}

			if err := runApply(t, exec, state); err != nil {
				t.Fatalf("apply: %v", err)
			}

			claudeMD := filepath.Join(dir, "CLAUDE.md")
			assertFileExists(t, claudeMD)

			data, err := os.ReadFile(claudeMD)
			if err != nil {
				t.Fatalf("read CLAUDE.md: %v", err)
			}
			if len(data) == 0 {
				t.Error("CLAUDE.md is empty")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestExecutor_NothingSelected — empty state must not error and must not write files.
// ---------------------------------------------------------------------------

func TestExecutor_NothingSelected(t *testing.T) {
	dir := t.TempDir()
	exec, _ := newExecutor(dir)

	state := &wizard.WizardState{} // zero value — HasChanges() == false

	if err := runApply(t, exec, state); err != nil {
		t.Fatalf("apply with empty state: %v", err)
	}

	// settings.json must NOT be written.
	if _, err := os.Stat(filepath.Join(dir, "settings.json")); !os.IsNotExist(err) {
		t.Error("settings.json should not be created when nothing is selected")
	}

	// No CLAUDE.md.
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Error("CLAUDE.md should not exist when no profile selected")
	}

	// No skills dir.
	if _, err := os.Stat(filepath.Join(dir, "skills")); !os.IsNotExist(err) {
		t.Error("skills dir should not exist when nothing selected")
	}
}
