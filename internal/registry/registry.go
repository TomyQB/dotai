package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SchemaVersion is the current registry schema version. All writes emit this.
const SchemaVersion = 2

// Entry is a single project registered with mem-cli.
type Entry struct {
	Path          string    `json:"path"`
	Name          string    `json:"name"`
	InitializedAt time.Time `json:"initialized_at"`
	LastSeen      time.Time `json:"last_seen"`
}

// Registry is the in-memory representation of the registry file.
// On disk the schema is versioned; Load() migrates v1 files transparently,
// Save() always emits v2.
type Registry struct {
	Version  int     `json:"version"`
	Projects []Entry `json:"projects"`
}

// RegistryError wraps a registry read/parse failure with context about
// the offending path. Callers can use errors.As to unwrap.
type RegistryError struct {
	Path string
	Err  error
}

func (e *RegistryError) Error() string {
	return fmt.Sprintf("registry: %s: %v", e.Path, e.Err)
}

func (e *RegistryError) Unwrap() error { return e.Err }

// Load reads the registry from disk. A missing file is not an error: it yields
// an empty v2 registry. Malformed JSON returns a *RegistryError.
func Load() (*Registry, error) {
	path := Path()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Registry{Version: SchemaVersion}, nil
		}
		return nil, &RegistryError{Path: path, Err: err}
	}
	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, &RegistryError{Path: path, Err: err}
	}
	if reg.Version == 0 {
		reg.Version = SchemaVersion
	}
	return &reg, nil
}

// Save writes the registry to disk in v2 shape using an atomic tmp-file +
// rename. The parent directory is created if needed. Permissions are 0644.
func (r *Registry) Save() error {
	if r == nil {
		return errors.New("registry: Save on nil receiver")
	}
	// Always emit v2 regardless of how we were loaded.
	r.Version = SchemaVersion
	if r.Projects == nil {
		r.Projects = []Entry{}
	}
	path := Path()
	if err := ensureParentDir(path); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("registry: marshal: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".projects-*.json.tmp")
	if err != nil {
		return fmt.Errorf("registry: create tmp: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("registry: write tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("registry: close tmp: %w", err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		cleanup()
		return fmt.Errorf("registry: chmod tmp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return fmt.Errorf("registry: rename: %w", err)
	}
	return nil
}

// Prune removes entries whose Path does not exist on disk. Returns the number
// of entries removed. Does NOT persist — callers decide when to Save.
func (r *Registry) Prune() int {
	if r == nil {
		return 0
	}
	kept := make([]Entry, 0, len(r.Projects))
	removed := 0
	for _, e := range r.Projects {
		if _, err := os.Stat(e.Path); err == nil {
			kept = append(kept, e)
		} else {
			removed++
		}
	}
	r.Projects = kept
	return removed
}

// Touch upserts an entry for path. New entries get both timestamps set to now.
// Existing entries only have LastSeen updated; InitializedAt is preserved.
func (r *Registry) Touch(path string) {
	if r == nil {
		return
	}
	now := time.Now().UTC()
	for i := range r.Projects {
		if r.Projects[i].Path == path {
			r.Projects[i].LastSeen = now
			if r.Projects[i].Name == "" {
				r.Projects[i].Name = filepath.Base(path)
			}
			return
		}
	}
	r.Projects = append(r.Projects, Entry{
		Path:          path,
		Name:          filepath.Base(path),
		InitializedAt: now,
		LastSeen:      now,
	})
}

// Find returns the entry matching path, if any.
func (r *Registry) Find(path string) (*Entry, bool) {
	if r == nil {
		return nil, false
	}
	for i := range r.Projects {
		if r.Projects[i].Path == path {
			return &r.Projects[i], true
		}
	}
	return nil, false
}
