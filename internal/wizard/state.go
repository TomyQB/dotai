// Package wizard implements the dotai setup wizard: an interactive TUI that
// guides the user through configuring Claude Code on their machine.
package wizard

import "github.com/TomyQB/dotai/internal/provider"

// Phase enumerates the top-level lifecycle stages of the wizard.
type Phase int

const (
	// PhaseStepping is the main interview phase where steps are presented.
	PhaseStepping Phase = iota
	// PhaseSummary shows the summary screen before applying changes.
	PhaseSummary
	// PhaseApplying executes the pending changes on disk.
	PhaseApplying
	// PhaseDone shows a success screen after all changes are applied.
	PhaseDone
	// PhaseError shows an error screen when something goes wrong.
	PhaseError
)

// ProfileType represents the type of CLAUDE.md profile to install.
type ProfileType int

const (
	// ProfileNone means no profile is installed.
	ProfileNone ProfileType = iota
	// ProfilePersonal installs the personal profile.
	ProfilePersonal
	// ProfileWork installs the work profile.
	ProfileWork
)

// String returns the human-readable name of the profile type.
func (p ProfileType) String() string {
	switch p {
	case ProfilePersonal:
		return "personal"
	case ProfileWork:
		return "work"
	default:
		return "none"
	}
}

// WizardState holds the accumulated answers from all wizard steps.
// It is passed (by pointer) through every step so each one can read
// previous answers and record its own.
type WizardState struct {
	// Provider is the AI tool backend being configured.
	Provider provider.Provider

	// SettingsEnabled indicates whether the user wants base settings applied.
	SettingsEnabled bool

	// StatuslineEnabled indicates whether the user wants shell status-line
	// integration installed.
	StatuslineEnabled bool

	// GitBranchHookEnabled indicates whether the user wants the git branch
	// check hook installed.
	GitBranchHookEnabled bool

	// ProfileType is the profile the user selected (or ProfileNone).
	ProfileType ProfileType

	// SelectedSkills contains the names of skills the user chose to install.
	SelectedSkills []string

	// MemcliEnabled indicates whether the user wants mem-cli hooks installed.
	MemcliEnabled bool

	// Patch holds the concrete changes that the executor will apply.
	Patch SettingsPatch
}

// HasChanges reports whether any installation action was selected.
func (s *WizardState) HasChanges() bool {
	return s.SettingsEnabled ||
		s.StatuslineEnabled ||
		s.GitBranchHookEnabled ||
		s.ProfileType != ProfileNone ||
		len(s.SelectedSkills) > 0 ||
		s.MemcliEnabled
}

// SettingsPatch describes all the changes the executor should apply.
type SettingsPatch struct {
	// BaseSettings is the parsed content of the settings template, ready to
	// be deep-merged into the existing settings.json.
	BaseSettings map[string]any

	// StatusLine holds the status-line configuration to install.
	StatusLine StatusLineConfig

	// Hooks is the list of hook entries to add to settings.json.
	Hooks []HookEntry
}

// StatusLineConfig holds the parameters for the shell status-line integration.
type StatusLineConfig struct {
	// Command is the shell command/script path to register.
	Command string
}

// HookEntry represents a single hook to be added to the provider's settings.
type HookEntry struct {
	// Event is the hook lifecycle event (e.g. "PreToolUse", "Stop").
	Event string
	// Matcher is an optional tool/event matcher expression.
	Matcher string
	// Command is the shell command to execute when the hook fires.
	Command string
}
