// Package localprofile installs bundled skill sets (and optional CLAUDE.md
// templates) into a target project directory's .claude/ folder. Used by the
// `dotai local` command to configure the repository where it is invoked.
package localprofile

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/TomyQB/dotai/internal/assets"
)

// Profile describes a bundle of skills (and an optional CLAUDE.md template)
// that can be installed into a project's .claude/ directory.
type Profile struct {
	Name          string
	DisplayName   string
	Description   string
	Skills        []string
	ClaudeMdAsset string
}

// All returns every profile available to `dotai local`, in display order.
func All() []Profile {
	return []Profile{Java, Web3}
}

// Java installs the Spring-Boot-oriented skill bundle. It does not write a
// CLAUDE.md — per product decision, only Web3 ships a project CLAUDE.md.
var Java = Profile{
	Name:        "java",
	DisplayName: "Java",
	Description: "Spring Boot, testing, OWASP and Postman skills",
	Skills: []string{
		"java-review",
		"java-spring-boot",
		"junit-mockito",
		"owasp-audit",
		"postman-generator",
	},
}

// Web3 installs the Foundry/Solidity skill bundle plus the project CLAUDE.md
// that encodes the Solidity style guide.
var Web3 = Profile{
	Name:        "web3",
	DisplayName: "Web3",
	Description: "Foundry, audit, review and Pinata skills + CLAUDE.md",
	Skills: []string{
		"web3-audit",
		"web3-review",
		"pinata-nft-upload",
	},
	ClaudeMdAsset: "files/local-profiles/web3/CLAUDE.md",
}

// SkillsDir returns the on-disk skills directory for the given target.
func SkillsDir(targetDir string) string {
	return filepath.Join(targetDir, ".claude", "skills")
}

// ClaudeMdPath returns the on-disk CLAUDE.md path for the given target.
func ClaudeMdPath(targetDir string) string {
	return filepath.Join(targetDir, ".claude", "CLAUDE.md")
}

// IsSkillInstalled reports whether a single skill directory exists in the target.
func IsSkillInstalled(targetDir, skill string) bool {
	_, err := os.Stat(filepath.Join(SkillsDir(targetDir), skill))
	return err == nil
}

// IsInstalled reports whether any artifact of the profile is present in the
// target. For Web3 that includes the CLAUDE.md template; for Java, only skills.
// Partial presence counts as installed so Uninstall can clean it up.
func (p Profile) IsInstalled(targetDir string) bool {
	for _, skill := range p.Skills {
		if IsSkillInstalled(targetDir, skill) {
			return true
		}
	}
	if p.ClaudeMdAsset != "" {
		if _, err := os.Stat(ClaudeMdPath(targetDir)); err == nil {
			return true
		}
	}
	return false
}

// Install copies the profile's skills (and CLAUDE.md, if any) into
// targetDir/.claude/. Existing files are overwritten.
func (p Profile) Install(targetDir string) error {
	skillsDst := SkillsDir(targetDir)
	if err := os.MkdirAll(skillsDst, 0755); err != nil {
		return err
	}
	for _, skill := range p.Skills {
		src := "files/skills/" + skill
		dst := filepath.Join(skillsDst, skill)
		if err := copyTree(assets.FS, src, dst); err != nil {
			return err
		}
	}
	if p.ClaudeMdAsset != "" {
		dst := ClaudeMdPath(targetDir)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		if err := copyAsset(assets.FS, p.ClaudeMdAsset, dst, 0644); err != nil {
			return err
		}
	}
	return nil
}

// Uninstall removes every skill directory that belongs to the profile, and
// the CLAUDE.md template if the profile shipped one. Missing paths are ignored.
func (p Profile) Uninstall(targetDir string) error {
	skillsDst := SkillsDir(targetDir)
	for _, skill := range p.Skills {
		if err := os.RemoveAll(filepath.Join(skillsDst, skill)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if p.ClaudeMdAsset != "" {
		if err := os.Remove(ClaudeMdPath(targetDir)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// copyTree mirrors an fs.FS subtree into a real filesystem destination.
// Duplicated here (rather than reused from internal/wizard) to keep the
// local-mode package self-contained and free of wizard-specific coupling.
func copyTree(fsys fs.FS, srcDir, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}
	return fs.WalkDir(fsys, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == srcDir {
			return nil
		}
		rel, _ := filepath.Rel(srcDir, path)
		dest := filepath.Join(dstDir, rel)

		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}

		perm := fs.FileMode(0644)
		if strings.HasSuffix(path, ".sh") {
			perm = 0755
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, data, perm)
	})
}

// copyAsset reads a single embedded file and writes it to dst.
func copyAsset(fsys fs.FS, src, dst string, perm os.FileMode) error {
	data, err := fs.ReadFile(fsys, src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, perm)
}
