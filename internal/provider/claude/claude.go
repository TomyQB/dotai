// Package claude implements the Provider interface for Claude Code.
// Configuration lives under ~/.claude/.
package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/TomyQB/dotai/internal/provider"
)

// Provider implements provider.Provider for Claude Code.
type Provider struct {
	mu         sync.Once
	configDir  string
	configErr  error
}

// New returns a new Claude Code provider.
func New() *Provider {
	return &Provider{}
}

// Name returns the human-readable provider name.
func (p *Provider) Name() string {
	return "Claude Code"
}

// ConfigDir returns ~/.claude, creating it if needed.
// The result is cached after the first successful call.
func (p *Provider) ConfigDir() (string, error) {
	p.mu.Do(func() {
		home, err := os.UserHomeDir()
		if err != nil {
			p.configErr = fmt.Errorf("claude: resolve home dir: %w", err)
			return
		}
		dir := filepath.Join(home, ".claude")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			p.configErr = fmt.Errorf("claude: create config dir %s: %w", dir, err)
			return
		}
		p.configDir = dir
	})
	return p.configDir, p.configErr
}

// SettingsFile returns the base filename of the Claude Code settings file.
func (p *Provider) SettingsFile() string {
	return "settings.json"
}

// SkillsDir returns the relative sub-path for skills within ConfigDir.
func (p *Provider) SkillsDir() string {
	return "skills"
}

// AgentsDir returns the relative sub-path for agent files within ConfigDir.
func (p *Provider) AgentsDir() string {
	return "agents"
}

// ToolDir returns the relative sub-path for tool-specific files within ConfigDir.
func (p *Provider) ToolDir() string {
	return "tools"
}

// ReadSettings reads and parses ~/.claude/settings.json.
// Returns an empty map when the file does not exist.
// Returns an error when the file exists but contains invalid JSON.
func (p *Provider) ReadSettings() (map[string]any, error) {
	dir, err := p.ConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, p.SettingsFile())

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("claude: read settings: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("claude: parse settings: %w", err)
	}
	return result, nil
}

// WriteSettings atomically writes data as indented JSON to ~/.claude/settings.json.
// The write is atomic: a temporary file in the same directory is written first,
// then renamed over the target to avoid partial writes.
func (p *Provider) WriteSettings(data map[string]any) error {
	dir, err := p.ConfigDir()
	if err != nil {
		return err
	}

	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("claude: marshal settings: %w", err)
	}
	// Append trailing newline for clean git diffs.
	encoded = append(encoded, '\n')

	// Write to a temp file in the same directory so the rename is atomic.
	tmp, err := os.CreateTemp(dir, ".settings-*.json.tmp")
	if err != nil {
		return fmt.Errorf("claude: create temp settings file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(encoded); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("claude: write temp settings: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("claude: close temp settings: %w", err)
	}

	target := filepath.Join(dir, p.SettingsFile())
	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("claude: rename settings: %w", err)
	}
	return nil
}

// SupportsFeature reports whether the provider supports the given feature.
// Claude Code v0.1 supports all defined features.
func (p *Provider) SupportsFeature(_ provider.Feature) bool {
	return true
}
