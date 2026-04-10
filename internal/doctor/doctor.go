// Package doctor inspects a mem-cli installation and a project repository for
// configuration issues. Inspect is a pure function: it returns a Report with
// no side effects. CLI rendering is separated into Report.String().
package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TomyQB/mem-cli/internal/registry"
)

// Status is the result of a single check.
type Status int

const (
	// StatusOK means the check passed.
	StatusOK Status = iota
	// StatusWarn means the check produced a warning (reserved; currently unused
	// but part of the public contract so future checks can use it without a
	// breaking signature change).
	StatusWarn
	// StatusFail means the check failed and the user should act on it.
	StatusFail
)

// Check is a single inspection result.
type Check struct {
	Name   string
	Status Status
	Detail string
}

// Report is the full result of Inspect for a given repo path. It contains
// installation-level (Global) checks and, when the cwd is a git repo,
// project-level checks.
type Report struct {
	RepoPath string
	Global   []Check
	Project  []Check
}

// HasFailures reports whether any check in the report has StatusFail.
func (r Report) HasFailures() bool {
	for _, c := range r.Global {
		if c.Status == StatusFail {
			return true
		}
	}
	for _, c := range r.Project {
		if c.Status == StatusFail {
			return true
		}
	}
	return false
}

// String renders the Report as the canonical human-readable CLI output.
// The format is preserved across refactors; the golden test in doctor_test.go
// locks the exact byte layout.
func (r Report) String() string {
	var b strings.Builder
	all := append([]Check{}, r.Global...)
	all = append(all, r.Project...)
	fail := 0
	for _, c := range all {
		mark := "✓"
		if c.Status == StatusFail {
			mark = "✗"
			fail++
		}
		fmt.Fprintf(&b, "  %s  %s\n", mark, c.Name)
		if c.Status == StatusFail && c.Detail != "" {
			fmt.Fprintf(&b, "      → %s\n", c.Detail)
		}
	}
	b.WriteString("\n")
	if fail == 0 {
		b.WriteString("All checks passed.\n")
	} else {
		fmt.Fprintf(&b, "%d check(s) failed.\n", fail)
	}
	return b.String()
}

// Inspect runs all configured checks and returns a Report.
// It is pure: no output is written, no process exits. Any error from
// environment resolution (e.g. no HOME) surfaces as a single failed check
// so the caller can render it uniformly.
func Inspect(repoPath string) Report {
	report := Report{RepoPath: repoPath}

	home, err := os.UserHomeDir()
	if err != nil {
		report.Global = append(report.Global, Check{
			Name:   "user home dir",
			Status: StatusFail,
			Detail: err.Error(),
		})
		return report
	}
	claude := filepath.Join(home, ".claude")

	report.Global = []Check{
		fileCheck("skill memcli-init", filepath.Join(claude, "skills/memcli-init/SKILL.md")),
		fileCheck("skill memcli-update", filepath.Join(claude, "skills/memcli-update/SKILL.md")),
		fileCheck("skill memcli-scan", filepath.Join(claude, "skills/memcli-scan/SKILL.md")),
		fileCheck("skill memcli-doctor", filepath.Join(claude, "skills/memcli-doctor/SKILL.md")),
		fileCheck("agent doc-keeper", filepath.Join(claude, "agents/doc-keeper.md")),
		fileCheck("hook stop-hook.sh", filepath.Join(claude, "mem-cli/hooks/stop-hook.sh")),
		fileCheck("hook pre-commit.sh", filepath.Join(claude, "mem-cli/hooks/pre-commit.sh")),
		settingsCheck(filepath.Join(claude, "settings.json")),
	}

	if isGitRepo(repoPath) {
		report.Project = []Check{
			fileCheck("project .agent-memory/index.md", filepath.Join(repoPath, ".agent-memory/index.md")),
			projectHookCheck(repoPath),
			registryCheck(repoPath),
			staleCheck(repoPath),
		}
	}
	return report
}

func fileCheck(name, path string) Check {
	if _, err := os.Stat(path); err == nil {
		return Check{Name: name, Status: StatusOK}
	}
	return Check{
		Name:   name,
		Status: StatusFail,
		Detail: "missing: " + path + "  (run: memcli install)",
	}
}

func settingsCheck(path string) Check {
	data, err := os.ReadFile(path)
	if err != nil {
		return Check{
			Name:   "settings.json Stop hook",
			Status: StatusFail,
			Detail: "missing settings.json — run: memcli install",
		}
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return Check{Name: "settings.json Stop hook", Status: StatusFail, Detail: "invalid JSON"}
	}
	hooks, _ := root["hooks"].(map[string]any)
	stop, _ := hooks["Stop"].([]any)
	for _, e := range stop {
		m, _ := e.(map[string]any)
		hs, _ := m["hooks"].([]any)
		for _, h := range hs {
			hm, _ := h.(map[string]any)
			cmd, _ := hm["command"].(string)
			if strings.Contains(cmd, "mem-cli/hooks/stop-hook") {
				return Check{Name: "settings.json Stop hook", Status: StatusOK}
			}
		}
	}
	return Check{Name: "settings.json Stop hook", Status: StatusFail, Detail: "run: memcli install"}
}

func isGitRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func projectHookCheck(repo string) Check {
	path := filepath.Join(repo, ".git/hooks/pre-commit")
	data, err := os.ReadFile(path)
	if err != nil {
		return Check{
			Name:   "project pre-commit hook",
			Status: StatusFail,
			Detail: "run /memcli-init inside Claude Code",
		}
	}
	if !strings.Contains(string(data), "agent-memory") {
		return Check{
			Name:   "project pre-commit hook",
			Status: StatusFail,
			Detail: "pre-commit exists but is not mem-cli's — merge manually",
		}
	}
	return Check{Name: "project pre-commit hook", Status: StatusOK}
}

// registryCheck uses internal/registry as the single source of truth.
// It reports OK if the current repo path is present in the registry.
func registryCheck(repo string) Check {
	reg, err := registry.Load()
	if err != nil {
		return Check{
			Name:   "project in registry",
			Status: StatusFail,
			Detail: "invalid registry: " + err.Error(),
		}
	}
	if _, ok := reg.Find(repo); ok {
		return Check{Name: "project in registry", Status: StatusOK}
	}
	return Check{
		Name:   "project in registry",
		Status: StatusFail,
		Detail: "run /memcli-init",
	}
}

func staleCheck(repo string) Check {
	path := filepath.Join(repo, ".agent-memory/.stale")
	data, err := os.ReadFile(path)
	if err != nil {
		return Check{Name: ".stale status", Status: StatusOK}
	}
	lines := 0
	for _, l := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(l) != "" {
			lines++
		}
	}
	if lines == 0 {
		return Check{Name: ".stale status (empty)", Status: StatusOK}
	}
	return Check{
		Name:   fmt.Sprintf(".stale status (%d pending)", lines),
		Status: StatusFail,
		Detail: "run /memcli-update inside Claude Code",
	}
}
