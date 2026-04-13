package wizard_test

import (
	"testing"

	"github.com/TomyQB/dotai/internal/wizard"
)

// TestHasChanges verifies that HasChanges returns true when any installation
// action is selected and false when the state is entirely empty.
func TestHasChanges(t *testing.T) {
	tests := []struct {
		name  string
		state wizard.WizardState
		want  bool
	}{
		{
			name:  "empty state has no changes",
			state: wizard.WizardState{},
			want:  false,
		},
		{
			name:  "SettingsEnabled triggers changes",
			state: wizard.WizardState{SettingsEnabled: true},
			want:  true,
		},
		{
			name:  "StatuslineEnabled triggers changes",
			state: wizard.WizardState{StatuslineEnabled: true},
			want:  true,
		},
		{
			name:  "GitBranchHookEnabled triggers changes",
			state: wizard.WizardState{GitBranchHookEnabled: true},
			want:  true,
		},
		{
			name:  "ProfilePersonal triggers changes",
			state: wizard.WizardState{ProfileType: wizard.ProfilePersonal},
			want:  true,
		},
		{
			name:  "ProfileWork triggers changes",
			state: wizard.WizardState{ProfileType: wizard.ProfileWork},
			want:  true,
		},
		{
			name:  "ProfileNone does not trigger changes",
			state: wizard.WizardState{ProfileType: wizard.ProfileNone},
			want:  false,
		},
		{
			name:  "SelectedSkills non-empty triggers changes",
			state: wizard.WizardState{SelectedSkills: []string{"commit-and-push"}},
			want:  true,
		},
		{
			name:  "SelectedSkills empty slice does not trigger changes",
			state: wizard.WizardState{SelectedSkills: []string{}},
			want:  false,
		},
		{
			name:  "MemcliEnabled triggers changes",
			state: wizard.WizardState{MemcliEnabled: true},
			want:  true,
		},
		{
			name: "all fields set triggers changes",
			state: wizard.WizardState{
				SettingsEnabled:      true,
				StatuslineEnabled:    true,
				GitBranchHookEnabled: true,
				ProfileType:          wizard.ProfileWork,
				SelectedSkills:       []string{"owasp-audit"},
				MemcliEnabled:        true,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.HasChanges()
			if got != tt.want {
				t.Errorf("HasChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestProfileTypeString verifies the human-readable string for each ProfileType.
func TestProfileTypeString(t *testing.T) {
	tests := []struct {
		profileType wizard.ProfileType
		want        string
	}{
		{wizard.ProfileNone, "none"},
		{wizard.ProfilePersonal, "personal"},
		{wizard.ProfileWork, "work"},
		// Unknown values fall through to "none".
		{wizard.ProfileType(99), "none"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.profileType.String()
			if got != tt.want {
				t.Errorf("ProfileType(%d).String() = %q, want %q", tt.profileType, got, tt.want)
			}
		})
	}
}
