// Package wizard — Update re-applies embedded templates and assets to the
// components the user already has installed, without adding anything new.
package wizard

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"text/template"

	"github.com/TomyQB/dotai/internal/assets"
	"github.com/TomyQB/dotai/internal/detect"
	"github.com/TomyQB/dotai/internal/provider"
)

// UpdateReport describes what Update touched and what it intentionally skipped.
type UpdateReport struct {
	// Updated is the list of component labels that were re-applied.
	Updated []string
	// Skipped is the list of {label, reason} pairs for components not touched.
	Skipped []UpdateSkip
}

// UpdateSkip describes a component that was not updated and why.
type UpdateSkip struct {
	Component string
	Reason    string
}

// Update detects every installed component and re-applies the embedded assets
// and settings template to each. Components that are not installed are left
// untouched — Update never introduces new components. Memcli is handled via
// its standalone install path because it merges a Stop hook that Update's
// synthetic WizardState does not carry.
func Update(prov provider.Provider) (UpdateReport, error) {
	report := UpdateReport{}

	configDir, err := prov.ConfigDir()
	if err != nil {
		return report, err
	}

	results := detect.CheckAll(prov)
	installedSet := make(map[detect.Component]detect.CheckResult, len(results))
	for _, r := range results {
		installedSet[r.Component] = r
	}

	state := &WizardState{Provider: prov}

	// Profile (CLAUDE.md) — overwrite only if the stored marker tells us which
	// variant the user picked. Legacy installs without the marker are reported
	// as skipped so the user knows to re-run Install to migrate.
	profileRes := installedSet[detect.Profile]
	var profileVariant ProfileType
	if profileRes.Installed {
		profileVariant = readProfileMarker(prov)
		if profileVariant != ProfileNone {
			state.ProfileType = profileVariant
			report.Updated = append(report.Updated, "CLAUDE.md ("+profileVariant.String()+")")
		} else {
			report.Skipped = append(report.Skipped, UpdateSkip{
				Component: "CLAUDE.md",
				Reason:    "unknown variant — run Install to set the profile",
			})
		}
	}

	// Settings BaseSettings merge — only if the full profile was installed
	// (i.e., CLAUDE.md present). Users who only installed memcli get a minimal
	// settings.json and must not receive the opinionated template.
	if profileRes.Installed {
		if base := loadSettingsTemplate(configDir, profileVariant); base != nil {
			state.SettingsEnabled = true
			state.Patch.BaseSettings = base
			report.Updated = append(report.Updated, "settings.json (base template)")
		} else {
			report.Skipped = append(report.Skipped, UpdateSkip{
				Component: "settings.json",
				Reason:    "could not load embedded template",
			})
		}
	}

	// Statusline script + settings entry — only if the script exists on disk.
	if installedSet[detect.Statusline].Installed {
		state.StatuslineEnabled = true
		state.Patch.StatusLine = StatusLineConfig{
			Command: "bash " + configDir + "/statusline-command.sh",
		}
		report.Updated = append(report.Updated, "statusline")
	}

	// Git branch-check hook — only if the script exists on disk.
	if installedSet[detect.GitHook].Installed {
		state.GitBranchHookEnabled = true
		state.Patch.Hooks = append(state.Patch.Hooks, HookEntry{
			Event:   "PreToolUse",
			Matcher: "Edit|Write|Agent",
			Command: "bash " + configDir + "/" + prov.ToolDir() + "/hooks/git-branch-check.sh",
		})
		report.Updated = append(report.Updated, "git-branch-check hook")
	}

	// Skills — only re-copy those that are present on disk.
	installedSkills, skillsErr := detect.InstalledSkills(prov)
	if skillsErr == nil && len(installedSkills) > 0 {
		state.SelectedSkills = installedSkills
		label := "skills"
		if len(installedSkills) == 1 {
			label = "skill"
		}
		report.Updated = append(report.Updated, fmt.Sprintf("%d %s", len(installedSkills), label))
	}

	// Apply everything the synthetic state captured (all non-memcli).
	if stateHasWork(state) {
		exec := NewExecutor(prov)
		if err := exec.apply(state); err != nil {
			return report, err
		}
	}

	// Memcli handled separately because InstallMemcli owns the Stop hook merge.
	if installedSet[detect.Memcli].Installed {
		if err := InstallMemcli(prov); err != nil {
			return report, err
		}
		report.Updated = append(report.Updated, "memcli (skills + agent + hooks)")
	}

	return report, nil
}

// readProfileMarker reads settings.json and returns the profile variant stored
// under the "_dotai.profile" key. Returns ProfileNone when the marker is
// missing, unreadable, or has an unknown value.
func readProfileMarker(prov provider.Provider) ProfileType {
	existing, err := prov.ReadSettings()
	if err != nil {
		return ProfileNone
	}
	rawDotai, ok := existing["_dotai"].(map[string]any)
	if !ok {
		return ProfileNone
	}
	profile, _ := rawDotai["profile"].(string)
	switch profile {
	case "personal":
		return ProfilePersonal
	case "work":
		return ProfileWork
	default:
		return ProfileNone
	}
}

// loadSettingsTemplate parses the embedded settings template, interpolates the
// config dir, and re-injects the _dotai.profile marker so the Update path
// does not accidentally drop it during the deep merge.
func loadSettingsTemplate(configDir string, variant ProfileType) map[string]any {
	raw, err := fs.ReadFile(assets.FS, "files/settings/settings.json.tmpl")
	if err != nil {
		return nil
	}
	tmpl, err := template.New("settings").Parse(string(raw))
	if err != nil {
		return nil
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, struct{ ClaudeDir string }{ClaudeDir: configDir}); err != nil {
		return nil
	}
	var base map[string]any
	if err := json.Unmarshal([]byte(sb.String()), &base); err != nil {
		return nil
	}
	if variant != ProfileNone {
		base["_dotai"] = map[string]any{"profile": variant.String()}
	}
	return base
}

// stateHasWork reports whether the synthetic state has at least one flag set
// that the executor would act on.
func stateHasWork(s *WizardState) bool {
	return s.SettingsEnabled ||
		s.StatuslineEnabled ||
		s.GitBranchHookEnabled ||
		s.ProfileType != ProfileNone ||
		len(s.SelectedSkills) > 0
}

