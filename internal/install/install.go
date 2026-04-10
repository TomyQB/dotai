package install

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/TomyQB/mem-cli/internal/assets"
)

// Run installs mem-cli skills, agents and hooks into ~/.claude/.
// Safe to re-run: overwrites mem-cli-owned files, merges settings.json.
func Run() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		return err
	}

	if err := copyTree("files/skills", filepath.Join(claudeDir, "skills")); err != nil {
		return fmt.Errorf("install skills: %w", err)
	}
	if err := copyTree("files/agents", filepath.Join(claudeDir, "agents")); err != nil {
		return fmt.Errorf("install agents: %w", err)
	}
	hooksDst := filepath.Join(claudeDir, "mem-cli", "hooks")
	if err := copyTree("files/hooks", hooksDst); err != nil {
		return fmt.Errorf("install hooks: %w", err)
	}
	// make hook scripts executable
	if err := chmodExec(hooksDst); err != nil {
		return err
	}

	if err := mergeSettings(filepath.Join(claudeDir, "settings.json"), hooksDst); err != nil {
		return fmt.Errorf("merge settings: %w", err)
	}

	fmt.Println("✓ mem-cli installed")
	fmt.Println("  skills:  ~/.claude/skills/memcli-{init,update,scan,doctor}")
	fmt.Println("  agent:   ~/.claude/agents/doc-keeper.md")
	fmt.Println("  hooks:   ~/.claude/mem-cli/hooks/{stop-hook.sh,pre-commit.sh}")
	fmt.Println("  Stop hook registered in ~/.claude/settings.json")
	fmt.Println()
	fmt.Println("Next: open Claude Code inside a project and run  /memcli-init")
	return nil
}

func copyTree(srcRoot, dstRoot string) error {
	return fs.WalkDir(assets.FS, srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(srcRoot, path)
		target := filepath.Join(dstRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := assets.FS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func chmodExec(dir string) error {
	return filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.HasSuffix(p, ".sh") {
			return os.Chmod(p, 0o755)
		}
		return nil
	})
}

// mergeSettings adds the mem-cli Stop hook to ~/.claude/settings.json without
// clobbering unrelated entries. It is idempotent.
func mergeSettings(path, hooksDir string) error {
	hookCmd := filepath.Join(hooksDir, "stop-hook.sh")

	var root map[string]any
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &root); err != nil {
			return fmt.Errorf("existing settings.json is not valid JSON: %w", err)
		}
	}
	if root == nil {
		root = map[string]any{}
	}

	hooks, _ := root["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	stopArr, _ := hooks["Stop"].([]any)

	// remove any prior mem-cli entries (idempotent)
	cleaned := stopArr[:0:0]
	for _, e := range stopArr {
		if !containsMemcli(e) {
			cleaned = append(cleaned, e)
		}
	}

	entry := map[string]any{
		"matcher": "",
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": hookCmd,
			},
		},
	}
	cleaned = append(cleaned, entry)
	hooks["Stop"] = cleaned
	root["hooks"] = hooks

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

func containsMemcli(e any) bool {
	m, ok := e.(map[string]any)
	if !ok {
		return false
	}
	hs, _ := m["hooks"].([]any)
	for _, h := range hs {
		hm, _ := h.(map[string]any)
		cmd, _ := hm["command"].(string)
		if strings.Contains(cmd, "mem-cli/hooks/stop-hook") {
			return true
		}
	}
	return false
}
