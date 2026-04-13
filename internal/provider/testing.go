// Package provider contains the TestProvider helper for use in tests.
package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// TestProvider is a Provider implementation backed by an arbitrary root
// directory (typically t.TempDir()). Use it in executor and wizard tests that
// need a real filesystem without touching ~/.claude.
type TestProvider struct {
	// Root is the base directory; all sub-paths are relative to it.
	Root string
}

// NewTestProvider returns a TestProvider whose config root is dir.
func NewTestProvider(dir string) *TestProvider {
	return &TestProvider{Root: dir}
}

// Name satisfies Provider.
func (p *TestProvider) Name() string { return "TestProvider" }

// ConfigDir satisfies Provider. Returns Root, creating it if absent.
func (p *TestProvider) ConfigDir() (string, error) {
	if err := os.MkdirAll(p.Root, 0o755); err != nil {
		return "", fmt.Errorf("testprovider: mkdir %s: %w", p.Root, err)
	}
	return p.Root, nil
}

// SettingsFile satisfies Provider.
func (p *TestProvider) SettingsFile() string { return "settings.json" }

// SkillsDir satisfies Provider.
func (p *TestProvider) SkillsDir() string { return "skills" }

// AgentsDir satisfies Provider.
func (p *TestProvider) AgentsDir() string { return "agents" }

// ToolDir satisfies Provider.
func (p *TestProvider) ToolDir() string { return "tools" }

// ReadSettings satisfies Provider. Returns empty map for missing file.
func (p *TestProvider) ReadSettings() (map[string]any, error) {
	path := filepath.Join(p.Root, p.SettingsFile())
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("testprovider: read settings: %w", err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("testprovider: parse settings: %w", err)
	}
	return result, nil
}

// WriteSettings satisfies Provider. Writes indented JSON with trailing newline.
func (p *TestProvider) WriteSettings(data map[string]any) error {
	dir, err := p.ConfigDir()
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("testprovider: marshal: %w", err)
	}
	encoded = append(encoded, '\n')
	path := filepath.Join(dir, p.SettingsFile())
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		return fmt.Errorf("testprovider: write: %w", err)
	}
	return nil
}

// SupportsFeature satisfies Provider. Always returns true in tests.
func (p *TestProvider) SupportsFeature(_ Feature) bool { return true }
