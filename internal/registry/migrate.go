package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
)

// rawRegistry is the minimum shape needed to sniff v1 vs v2. Projects decoded
// as []json.RawMessage so each element can be inspected individually for its
// type (string vs object).
type rawRegistry struct {
	Version  int               `json:"version"`
	Projects []json.RawMessage `json:"projects"`
}

// UnmarshalJSON accepts both registry shapes and produces a v2 Registry:
//
//	v1: {"projects": ["/a", "/b"]}
//	v2: {"version":2, "projects":[{"path":"/a","name":"a",...}]}
//
// Mixed arrays (some strings, some objects) are not supported and return an
// error on the first unexpected element.
func (r *Registry) UnmarshalJSON(data []byte) error {
	var raw rawRegistry
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Version = raw.Version
	r.Projects = make([]Entry, 0, len(raw.Projects))
	for i, item := range raw.Projects {
		trimmed := bytes.TrimSpace(item)
		if len(trimmed) == 0 {
			continue
		}
		switch trimmed[0] {
		case '"':
			var path string
			if err := json.Unmarshal(item, &path); err != nil {
				return fmt.Errorf("registry: projects[%d]: %w", i, err)
			}
			r.Projects = append(r.Projects, Entry{
				Path: path,
				Name: filepath.Base(path),
			})
		case '{':
			var e Entry
			if err := json.Unmarshal(item, &e); err != nil {
				return fmt.Errorf("registry: projects[%d]: %w", i, err)
			}
			if e.Name == "" && e.Path != "" {
				e.Name = filepath.Base(e.Path)
			}
			r.Projects = append(r.Projects, e)
		default:
			return fmt.Errorf("registry: projects[%d]: unexpected element kind", i)
		}
	}
	return nil
}
