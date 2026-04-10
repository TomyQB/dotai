// Package registry is the single source of truth for the mem-cli project registry.
// It handles reading and writing the registry file, transparent v1->v2 migration,
// and mutating operations (Touch, Prune).
package registry

import (
	"fmt"
	"os"
	"path/filepath"
)

// envOverride is the environment variable used to override the registry path
// for tests and debugging. When set, Path() returns its value verbatim.
const envOverride = "MEMCLI_REGISTRY"

// Path returns the absolute path to the registry file.
//
// Resolution order:
//  1. $MEMCLI_REGISTRY if set (testing / debug override)
//  2. $HOME/.claude/memcli/projects.json (canonical location per spec REQ-6)
//
// The path is NOT guaranteed to exist; use Load() which tolerates missing files.
func Path() string {
	if override := os.Getenv(envOverride); override != "" {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback: relative path. Callers will surface any downstream error.
		return filepath.Join(".claude", "memcli", "projects.json")
	}
	return filepath.Join(home, ".claude", "memcli", "projects.json")
}

// ensureParentDir creates the parent directory of path with 0755 permissions
// if it does not already exist. Returns any error from MkdirAll verbatim.
func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("registry: ensure parent dir %q: %w", dir, err)
	}
	return nil
}
