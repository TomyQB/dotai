package detect

import (
	"os"
	"path/filepath"

	"github.com/TomyQB/dotai/internal/assets"
	"github.com/TomyQB/dotai/internal/provider"
)

// InstalledSkills returns the subset of catalog skill directories that are
// currently present on disk. The catalog is derived from the embedded asset
// tree (files/skills/*) so it stays in sync with whatever the binary ships.
// Memcli skills (files/memcli/skills/*) are intentionally excluded — memcli
// has its own detection path via the Memcli component.
func InstalledSkills(prov provider.Provider) ([]string, error) {
	configDir, err := prov.ConfigDir()
	if err != nil {
		return nil, err
	}
	entries, err := assets.FS.ReadDir("files/skills")
	if err != nil {
		return nil, err
	}
	skillsRel := prov.SkillsDir()
	var installed []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		skillPath := filepath.Join(configDir, skillsRel, name)
		if _, err := os.Stat(skillPath); err == nil {
			installed = append(installed, name)
		}
	}
	return installed, nil
}
