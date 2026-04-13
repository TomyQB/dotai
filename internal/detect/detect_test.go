package detect_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TomyQB/dotai/internal/detect"
	"github.com/TomyQB/dotai/internal/provider"
)

func newProv(t *testing.T) *provider.TestProvider {
	t.Helper()
	return provider.NewTestProvider(t.TempDir())
}

// TestCheckAll_Order verifies that CheckAll returns 6 results in canonical
// component order: Settings, Statusline, GitHook, Profile, Skills, Memcli.
func TestCheckAll_Order(t *testing.T) {
	prov := newProv(t)
	results := detect.CheckAll(prov)

	if len(results) != 6 {
		t.Fatalf("expected 6 results, got %d", len(results))
	}

	want := []detect.Component{
		detect.Settings,
		detect.Statusline,
		detect.GitHook,
		detect.Profile,
		detect.Skills,
		detect.Memcli,
	}
	for i, r := range results {
		if r.Component != want[i] {
			t.Errorf("results[%d].Component = %v, want %v", i, r.Component, want[i])
		}
	}
}

// TestCheckComponent_Settings verifies present/absent detection for settings.json.
func TestCheckComponent_Settings(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		prov := newProv(t)
		dir, _ := prov.ConfigDir()
		if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
		r := detect.CheckComponent(prov, detect.Settings)
		if !r.Installed {
			t.Errorf("expected Installed=true, got false (detail: %s)", r.Detail)
		}
	})

	t.Run("absent", func(t *testing.T) {
		prov := newProv(t)
		r := detect.CheckComponent(prov, detect.Settings)
		if r.Installed {
			t.Errorf("expected Installed=false, got true (detail: %s)", r.Detail)
		}
	})
}

// TestCheckComponent_Statusline verifies present/absent detection for statusline.
func TestCheckComponent_Statusline(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		prov := newProv(t)
		dir, _ := prov.ConfigDir()
		if err := os.WriteFile(filepath.Join(dir, "statusline-command.sh"), []byte("#!/bin/sh"), 0o755); err != nil {
			t.Fatal(err)
		}
		r := detect.CheckComponent(prov, detect.Statusline)
		if !r.Installed {
			t.Errorf("expected Installed=true, got false (detail: %s)", r.Detail)
		}
	})

	t.Run("absent", func(t *testing.T) {
		prov := newProv(t)
		r := detect.CheckComponent(prov, detect.Statusline)
		if r.Installed {
			t.Errorf("expected Installed=false, got true (detail: %s)", r.Detail)
		}
	})
}

// TestCheckComponent_GitHook verifies present/absent detection for the git hook.
func TestCheckComponent_GitHook(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		prov := newProv(t)
		dir, _ := prov.ConfigDir()
		hookPath := filepath.Join(dir, prov.ToolDir(), "hooks", "git-branch-check.sh")
		if err := os.MkdirAll(filepath.Dir(hookPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(hookPath, []byte("#!/bin/sh"), 0o755); err != nil {
			t.Fatal(err)
		}
		r := detect.CheckComponent(prov, detect.GitHook)
		if !r.Installed {
			t.Errorf("expected Installed=true, got false (detail: %s)", r.Detail)
		}
	})

	t.Run("absent", func(t *testing.T) {
		prov := newProv(t)
		r := detect.CheckComponent(prov, detect.GitHook)
		if r.Installed {
			t.Errorf("expected Installed=false, got true (detail: %s)", r.Detail)
		}
	})
}

// TestCheckComponent_Profile verifies present/absent detection for CLAUDE.md.
func TestCheckComponent_Profile(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		prov := newProv(t)
		dir, _ := prov.ConfigDir()
		if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# profile"), 0o644); err != nil {
			t.Fatal(err)
		}
		r := detect.CheckComponent(prov, detect.Profile)
		if !r.Installed {
			t.Errorf("expected Installed=true, got false (detail: %s)", r.Detail)
		}
	})

	t.Run("absent", func(t *testing.T) {
		prov := newProv(t)
		r := detect.CheckComponent(prov, detect.Profile)
		if r.Installed {
			t.Errorf("expected Installed=false, got true (detail: %s)", r.Detail)
		}
	})
}

// TestCheckComponent_Skills verifies detection based on skills directory contents.
func TestCheckComponent_Skills(t *testing.T) {
	t.Run("dir with entries", func(t *testing.T) {
		prov := newProv(t)
		dir, _ := prov.ConfigDir()
		skillsDir := filepath.Join(dir, prov.SkillsDir())
		if err := os.MkdirAll(filepath.Join(skillsDir, "my-skill"), 0o755); err != nil {
			t.Fatal(err)
		}
		r := detect.CheckComponent(prov, detect.Skills)
		if !r.Installed {
			t.Errorf("expected Installed=true, got false (detail: %s)", r.Detail)
		}
	})

	t.Run("empty dir", func(t *testing.T) {
		prov := newProv(t)
		dir, _ := prov.ConfigDir()
		if err := os.MkdirAll(filepath.Join(dir, prov.SkillsDir()), 0o755); err != nil {
			t.Fatal(err)
		}
		r := detect.CheckComponent(prov, detect.Skills)
		if r.Installed {
			t.Errorf("expected Installed=false for empty dir, got true (detail: %s)", r.Detail)
		}
	})

	t.Run("absent", func(t *testing.T) {
		prov := newProv(t)
		r := detect.CheckComponent(prov, detect.Skills)
		if r.Installed {
			t.Errorf("expected Installed=false, got true (detail: %s)", r.Detail)
		}
	})
}

// TestCheckComponent_Memcli verifies present/absent/partial detection for memcli.
func TestCheckComponent_Memcli(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		prov := newProv(t)
		dir, _ := prov.ConfigDir()
		// Create all 3 required components.
		if err := os.MkdirAll(filepath.Join(dir, prov.SkillsDir(), "memcli-init"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, prov.AgentsDir()), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, prov.AgentsDir(), "doc-keeper.md"), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, prov.ToolDir(), "hooks"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, prov.ToolDir(), "hooks", "stop-hook.sh"), []byte("test"), 0o755); err != nil {
			t.Fatal(err)
		}
		r := detect.CheckComponent(prov, detect.Memcli)
		if !r.Installed {
			t.Errorf("expected Installed=true, got false (detail: %s)", r.Detail)
		}
	})

	t.Run("partial", func(t *testing.T) {
		prov := newProv(t)
		dir, _ := prov.ConfigDir()
		// Only create memcli-init skill (1 of 3).
		if err := os.MkdirAll(filepath.Join(dir, prov.SkillsDir(), "memcli-init"), 0o755); err != nil {
			t.Fatal(err)
		}
		r := detect.CheckComponent(prov, detect.Memcli)
		if r.Installed {
			t.Errorf("expected Installed=false for partial install, got true (detail: %s)", r.Detail)
		}
		if r.Detail == "not installed" {
			t.Errorf("expected partial detail, got %q", r.Detail)
		}
	})

	t.Run("absent", func(t *testing.T) {
		prov := newProv(t)
		r := detect.CheckComponent(prov, detect.Memcli)
		if r.Installed {
			t.Errorf("expected Installed=false, got true (detail: %s)", r.Detail)
		}
	})
}
