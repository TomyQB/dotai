// Package detect inspects the filesystem to determine which dotai components
// are currently installed for a given provider.
package detect

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/TomyQB/dotai/internal/provider"
)

// Component identifies a single detectable dotai component.
type Component int

const (
	// Settings represents the base settings.json configuration component.
	Settings Component = iota
	// Statusline represents the shell status-line integration component.
	Statusline
	// GitHook represents the git branch check hook component.
	GitHook
	// Profile represents the CLAUDE.md profile component.
	Profile
	// Skills represents the installed skills component.
	Skills
	// Memcli represents the mem-cli hooks and agent component.
	Memcli
)

// String returns the human-readable display label for the component.
func (c Component) String() string {
	switch c {
	case Settings:
		return "Settings"
	case Statusline:
		return "Statusline"
	case GitHook:
		return "Git Hook"
	case Profile:
		return "CLAUDE.md"
	case Skills:
		return "Skills"
	case Memcli:
		return "Memcli"
	default:
		return "Unknown"
	}
}

// CheckResult holds the detection outcome for a single component.
type CheckResult struct {
	// Component identifies which component was checked.
	Component Component
	// Installed reports whether the component is currently installed.
	Installed bool
	// Detail is a human-readable summary of the detection outcome.
	Detail string
}

// CheckAll inspects all components and returns results in canonical order:
// Settings, Statusline, GitHook, Profile, Skills, Memcli.
func CheckAll(prov provider.Provider) []CheckResult {
	components := []Component{Settings, Statusline, GitHook, Profile, Skills, Memcli}
	results := make([]CheckResult, len(components))
	for i, c := range components {
		results[i] = CheckComponent(prov, c)
	}
	return results
}

// CheckComponent inspects a single component and returns its CheckResult.
func CheckComponent(prov provider.Provider, c Component) CheckResult {
	configDir, err := prov.ConfigDir()
	if err != nil {
		return CheckResult{Component: c, Installed: false, Detail: "cannot resolve config dir: " + err.Error()}
	}

	switch c {
	case Settings:
		path := filepath.Join(configDir, "settings.json")
		if _, err := os.Stat(path); err == nil {
			return CheckResult{Component: Settings, Installed: true, Detail: "settings.json"}
		}
		return CheckResult{Component: Settings, Installed: false, Detail: "settings.json not found"}

	case Statusline:
		path := filepath.Join(configDir, "statusline-command.sh")
		if _, err := os.Stat(path); err == nil {
			return CheckResult{Component: Statusline, Installed: true, Detail: "statusline-command.sh"}
		}
		return CheckResult{Component: Statusline, Installed: false, Detail: "not found"}

	case GitHook:
		path := filepath.Join(configDir, prov.ToolDir(), "hooks", "git-branch-check.sh")
		if _, err := os.Stat(path); err == nil {
			return CheckResult{Component: GitHook, Installed: true, Detail: "git-branch-check.sh"}
		}
		return CheckResult{Component: GitHook, Installed: false, Detail: "not found"}

	case Profile:
		path := filepath.Join(configDir, "CLAUDE.md")
		if _, err := os.Stat(path); err == nil {
			return CheckResult{Component: Profile, Installed: true, Detail: "CLAUDE.md"}
		}
		return CheckResult{Component: Profile, Installed: false, Detail: "CLAUDE.md not found"}

	case Skills:
		skillsPath := filepath.Join(configDir, prov.SkillsDir())
		entries, err := os.ReadDir(skillsPath)
		if err == nil && len(entries) > 0 {
			return CheckResult{Component: Skills, Installed: true, Detail: fmt.Sprintf("%d skills installed", len(entries))}
		}
		return CheckResult{Component: Skills, Installed: false, Detail: "no skills"}

	case Memcli:
		// Check all key memcli components: skills, agent, and hooks.
		required := []string{
			filepath.Join(configDir, prov.SkillsDir(), "memcli-init"),
			filepath.Join(configDir, prov.AgentsDir(), "doc-keeper.md"),
			filepath.Join(configDir, prov.ToolDir(), "hooks", "stop-hook.sh"),
		}
		missing := 0
		for _, p := range required {
			if _, err := os.Stat(p); err != nil {
				missing++
			}
		}
		if missing == 0 {
			return CheckResult{Component: Memcli, Installed: true, Detail: "skills + agent + hooks"}
		}
		if missing < len(required) {
			return CheckResult{Component: Memcli, Installed: false, Detail: fmt.Sprintf("partial (%d/%d components)", len(required)-missing, len(required))}
		}
		return CheckResult{Component: Memcli, Installed: false, Detail: "not installed"}
	}

	return CheckResult{Component: c, Installed: false, Detail: "unknown component"}
}
