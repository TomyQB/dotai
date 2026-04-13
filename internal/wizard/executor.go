// Package wizard — Executor applies the pending changes described by WizardState
// to the filesystem and provider configuration.
package wizard

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/assets"
	"github.com/TomyQB/dotai/internal/provider"
)

// ProgressMsg is emitted by the Executor to report progress or completion.
type ProgressMsg struct {
	// Message is a human-readable status line.
	Message string
	// Done indicates all changes have been applied successfully.
	Done bool
	// Err holds the error if the executor failed.
	Err error
}

// PermissionError wraps a filesystem permission denial with the offending path.
type PermissionError struct {
	Path string
	Err  error
}

func (e *PermissionError) Error() string {
	return "permission denied: " + e.Path + ": " + e.Err.Error()
}

func (e *PermissionError) Unwrap() error { return e.Err }

// Executor applies the pending wizard changes to disk.
type Executor struct {
	prov provider.Provider
}

// NewExecutor constructs an Executor bound to the given provider.
func NewExecutor(prov provider.Provider) *Executor {
	return &Executor{prov: prov}
}

// Run starts the async application of changes and returns a Cmd that emits
// ProgressMsg values as work proceeds. The final message has Done=true or Err set.
func (e *Executor) Run(state *WizardState) tea.Cmd {
	return func() tea.Msg {
		if err := e.apply(state); err != nil {
			return ProgressMsg{Err: err}
		}
		return ProgressMsg{Message: "Done", Done: true}
	}
}

// apply performs all installation steps synchronously.
func (e *Executor) apply(state *WizardState) error {
	// Step 1: resolve config dir — abort on error.
	configDir, err := e.prov.ConfigDir()
	if err != nil {
		return err
	}

	// Step 2: ensure the config root exists.
	if err := mkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Step 3: create hooks dir if needed.
	if len(state.Patch.Hooks) > 0 {
		hooksDir := filepath.Join(configDir, e.prov.ToolDir(), "hooks")
		if err := mkdirAll(hooksDir, 0755); err != nil {
			return err
		}
	}

	// Step 4: create skills dir if needed.
	if len(state.SelectedSkills) > 0 {
		skillsDir := filepath.Join(configDir, e.prov.SkillsDir())
		if err := mkdirAll(skillsDir, 0755); err != nil {
			return err
		}
	}

	// Step 5: create agents dir if memcli is enabled.
	if state.MemcliEnabled {
		agentsDir := filepath.Join(configDir, e.prov.AgentsDir())
		if err := mkdirAll(agentsDir, 0755); err != nil {
			return err
		}
	}

	// Step 6: copy selected skills.
	for _, name := range state.SelectedSkills {
		src := "files/skills/" + name
		dst := filepath.Join(configDir, e.prov.SkillsDir(), name)
		if err := copyTree(assets.FS, src, dst); err != nil {
			return err
		}
	}

	// Step 7: copy memcli skills and agent file.
	if state.MemcliEnabled {
		// Walk memcli/skills and copy each skill subdir into the skills directory.
		skillsDstBase := filepath.Join(configDir, e.prov.SkillsDir())
		entries, err := fs.ReadDir(assets.FS, "files/memcli/skills")
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			src := "files/memcli/skills/" + entry.Name()
			dst := filepath.Join(skillsDstBase, entry.Name())
			if err := copyTree(assets.FS, src, dst); err != nil {
				return err
			}
		}

		// Copy the doc-keeper agent file.
		agentDst := filepath.Join(configDir, e.prov.AgentsDir(), "doc-keeper.md")
		if err := copyAsset(assets.FS, "files/memcli/agents/doc-keeper.md", agentDst, 0644); err != nil {
			return err
		}
	}

	// Step 8: copy git branch hook.
	if state.GitBranchHookEnabled {
		hooksDst := filepath.Join(configDir, e.prov.ToolDir(), "hooks")
		if err := copyTree(assets.FS, "files/hooks", hooksDst); err != nil {
			return err
		}
	}

	// Step 9: copy memcli hooks.
	if state.MemcliEnabled {
		hooksDst := filepath.Join(configDir, e.prov.ToolDir(), "hooks")
		if err := mkdirAll(hooksDst, 0755); err != nil {
			return err
		}
		entries, err := fs.ReadDir(assets.FS, "files/memcli-hooks")
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			src := "files/memcli-hooks/" + entry.Name()
			dst := filepath.Join(hooksDst, entry.Name())
			perm := fs.FileMode(0644)
			if strings.HasSuffix(entry.Name(), ".sh") {
				perm = 0755
			}
			if err := copyAsset(assets.FS, src, dst, perm); err != nil {
				return err
			}
		}
	}

	// Step 10: create memcli registry if needed.
	if state.MemcliEnabled {
		memcliDir := filepath.Join(configDir, "memcli")
		if err := mkdirAll(memcliDir, 0755); err != nil {
			return err
		}
		registryPath := filepath.Join(memcliDir, "projects.json")
		if _, err := os.Stat(registryPath); os.IsNotExist(err) {
			emptyRegistry := []byte(`{"version":2,"projects":[]}` + "\n")
			if err := writeFile(registryPath, emptyRegistry, 0644); err != nil {
				return err
			}
		}
	}

	// Step 11: copy statusline script.
	if state.StatuslineEnabled {
		dst := filepath.Join(configDir, "statusline-command.sh")
		if err := copyAsset(assets.FS, "files/statusline/statusline-command.sh", dst, 0755); err != nil {
			return err
		}
	}

	// Step 12: copy profile as CLAUDE.md.
	if state.ProfileType != ProfileNone {
		src := "files/profiles/" + state.ProfileType.String() + ".md"
		dst := filepath.Join(configDir, "CLAUDE.md")
		if err := copyAsset(assets.FS, src, dst, 0644); err != nil {
			return err
		}
	}

	// Step 13: merge and write settings.json last.
	settingsNeeded := state.SettingsEnabled ||
		state.StatuslineEnabled ||
		len(state.Patch.Hooks) > 0
	if settingsNeeded {
		if err := e.mergeAndWriteSettings(state); err != nil {
			return err
		}
	}

	return nil
}

// mergeAndWriteSettings merges wizard-configured values into the existing
// settings file and writes the result back via the provider.
func (e *Executor) mergeAndWriteSettings(state *WizardState) error {
	existing, err := e.prov.ReadSettings()
	if err != nil {
		existing = make(map[string]any)
	}

	// Merge base settings template.
	if state.SettingsEnabled && state.Patch.BaseSettings != nil {
		existing = deepMerge(existing, state.Patch.BaseSettings)
	}

	// Set statusLine entry.
	if state.StatuslineEnabled {
		existing["statusLine"] = map[string]any{
			"type":    "command",
			"command": state.Patch.StatusLine.Command,
		}
	}

	// Merge hook entries.
	for _, hook := range state.Patch.Hooks {
		// Ensure existing["hooks"] is a map[string]any.
		var hooksMap map[string]any
		if raw, ok := existing["hooks"]; ok {
			hooksMap, _ = raw.(map[string]any)
		}
		if hooksMap == nil {
			hooksMap = make(map[string]any)
		}

		// Get or create the event array.
		var eventSlice []any
		if raw, ok := hooksMap[hook.Event]; ok {
			eventSlice, _ = raw.([]any)
		}
		if eventSlice == nil {
			eventSlice = []any{}
		}

		// Filter out stale entries for the same script (dedup).
		// Extract the script basename from the hook command for matching.
		scriptBase := filepath.Base(hook.Command)
		filtered := make([]any, 0, len(eventSlice))
		for _, item := range eventSlice {
			if !hookCommandContains(item, scriptBase) {
				filtered = append(filtered, item)
			}
		}

		// Build new entry.
		newEntry := map[string]any{
			"hooks": []any{
				map[string]any{
					"type":    "command",
					"command": hook.Command,
				},
			},
		}
		if hook.Matcher != "" {
			newEntry["matcher"] = hook.Matcher
		}

		hooksMap[hook.Event] = append(filtered, newEntry)
		existing["hooks"] = hooksMap
	}

	return e.prov.WriteSettings(existing)
}

// deepMerge merges src into dst recursively. src values win on conflicts.
// dst is modified in place and returned for convenience.
func deepMerge(dst, src map[string]any) map[string]any {
	for k, srcVal := range src {
		dstVal, exists := dst[k]
		if exists {
			dstMap, dstIsMap := dstVal.(map[string]any)
			srcMap, srcIsMap := srcVal.(map[string]any)
			if dstIsMap && srcIsMap {
				dst[k] = deepMerge(dstMap, srcMap)
				continue
			}
		}
		dst[k] = srcVal
	}
	return dst
}

// copyTree recursively copies a directory tree from an fs.FS into the real filesystem.
func copyTree(fsys fs.FS, srcDir, dstDir string) error {
	// Ensure the destination root exists before writing any entries into it.
	if err := mkdirAll(dstDir, 0755); err != nil {
		return err
	}
	return fs.WalkDir(fsys, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip the root entry itself — dstDir was already created above.
		if path == srcDir {
			return nil
		}

		rel, _ := filepath.Rel(srcDir, path)
		dest := filepath.Join(dstDir, rel)

		if d.IsDir() {
			return mkdirAll(dest, 0755)
		}

		perm := fs.FileMode(0644)
		if strings.HasSuffix(path, ".sh") {
			perm = 0755
		}

		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		return writeFile(dest, data, perm)
	})
}

// copyAsset reads a single file from an fs.FS and writes it to dst.
func copyAsset(fsys fs.FS, src, dst string, perm os.FileMode) error {
	data, err := fs.ReadFile(fsys, src)
	if err != nil {
		return err
	}
	return writeFile(dst, data, perm)
}

// mkdirAll wraps os.MkdirAll, converting permission errors to PermissionError.
func mkdirAll(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		if os.IsPermission(err) {
			return &PermissionError{Path: path, Err: err}
		}
		return err
	}
	return nil
}

// writeFile wraps os.WriteFile, converting permission errors to PermissionError.
func writeFile(path string, data []byte, perm os.FileMode) error {
	if err := os.WriteFile(path, data, perm); err != nil {
		if os.IsPermission(err) {
			return &PermissionError{Path: path, Err: err}
		}
		return err
	}
	return nil
}

// InstallMemcli installs memcli components (skills, agent, hooks) independently,
// without requiring the full wizard flow.
func InstallMemcli(prov provider.Provider) error {
	configDir, err := prov.ConfigDir()
	if err != nil {
		return err
	}

	// Ensure required directories exist.
	skillsDir := filepath.Join(configDir, prov.SkillsDir())
	if err := mkdirAll(skillsDir, 0755); err != nil {
		return err
	}
	agentsDir := filepath.Join(configDir, prov.AgentsDir())
	if err := mkdirAll(agentsDir, 0755); err != nil {
		return err
	}
	hooksDir := filepath.Join(configDir, prov.ToolDir(), "hooks")
	if err := mkdirAll(hooksDir, 0755); err != nil {
		return err
	}

	// Copy memcli skills into the skills directory.
	entries, err := assets.FS.ReadDir("files/memcli/skills")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		src := "files/memcli/skills/" + entry.Name()
		dst := filepath.Join(skillsDir, entry.Name())
		if err := copyTree(assets.FS, src, dst); err != nil {
			return err
		}
	}

	// Copy the doc-keeper agent file.
	agentDst := filepath.Join(agentsDir, "doc-keeper.md")
	if err := copyAsset(assets.FS, "files/memcli/agents/doc-keeper.md", agentDst, 0644); err != nil {
		return err
	}

	// Copy memcli hooks into the hooks directory.
	hookEntries, err := assets.FS.ReadDir("files/memcli-hooks")
	if err != nil {
		return err
	}
	for _, entry := range hookEntries {
		if entry.IsDir() {
			continue
		}
		src := "files/memcli-hooks/" + entry.Name()
		dst := filepath.Join(hooksDir, entry.Name())
		perm := fs.FileMode(0644)
		if strings.HasSuffix(entry.Name(), ".sh") {
			perm = 0755
		}
		if err := copyAsset(assets.FS, src, dst, perm); err != nil {
			return err
		}
	}

	// Create the memcli registry directory and empty projects.json if absent.
	memcliDir := filepath.Join(configDir, "memcli")
	if err := mkdirAll(memcliDir, 0755); err != nil {
		return err
	}
	registryPath := filepath.Join(memcliDir, "projects.json")
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		emptyRegistry := []byte(`{"version":2,"projects":[]}` + "\n")
		if err := writeFile(registryPath, emptyRegistry, 0644); err != nil {
			return err
		}
	}

	// Merge stop hook into settings.json.
	existing, err := prov.ReadSettings()
	if err != nil {
		existing = make(map[string]any)
	}

	stopHookCmd := "bash " + configDir + "/" + prov.ToolDir() + "/hooks/stop-hook.sh"
	stopHook := HookEntry{Event: "Stop", Command: stopHookCmd}

	var hooksMap map[string]any
	if raw, ok := existing["hooks"]; ok {
		hooksMap, _ = raw.(map[string]any)
	}
	if hooksMap == nil {
		hooksMap = make(map[string]any)
	}

	var eventSlice []any
	if raw, ok := hooksMap[stopHook.Event]; ok {
		eventSlice, _ = raw.([]any)
	}
	if eventSlice == nil {
		eventSlice = []any{}
	}

	// Deduplicate existing stop-hook entries.
	filtered := make([]any, 0, len(eventSlice))
	for _, item := range eventSlice {
		if !hookCommandContains(item, "stop-hook.sh") {
			filtered = append(filtered, item)
		}
	}

	newEntry := map[string]any{
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": stopHookCmd,
			},
		},
	}
	hooksMap[stopHook.Event] = append(filtered, newEntry)
	existing["hooks"] = hooksMap

	return prov.WriteSettings(existing)
}

// UninstallMemcli removes memcli components from the provider configuration.
func UninstallMemcli(prov provider.Provider) error {
	configDir, err := prov.ConfigDir()
	if err != nil {
		return err
	}

	// Files to remove (ignore not-exist errors).
	filesToRemove := []string{
		filepath.Join(configDir, prov.ToolDir(), "hooks", "pre-commit.sh"),
		filepath.Join(configDir, prov.ToolDir(), "hooks", "stop-hook.sh"),
		filepath.Join(configDir, prov.AgentsDir(), "doc-keeper.md"),
	}
	for _, f := range filesToRemove {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	// Directories to remove (ignore not-exist errors).
	dirsToRemove := []string{
		filepath.Join(configDir, prov.SkillsDir(), "memcli-doctor"),
		filepath.Join(configDir, prov.SkillsDir(), "memcli-init"),
		filepath.Join(configDir, prov.SkillsDir(), "memcli-scan"),
		filepath.Join(configDir, prov.SkillsDir(), "memcli-update"),
		filepath.Join(configDir, "memcli"),
	}
	for _, d := range dirsToRemove {
		if err := os.RemoveAll(d); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	// Remove the stop hook entry from settings.json.
	existing, err := prov.ReadSettings()
	if err != nil {
		// If settings can't be read, nothing to clean up there.
		return nil //nolint:nilerr
	}

	var hooksMap map[string]any
	if raw, ok := existing["hooks"]; ok {
		hooksMap, _ = raw.(map[string]any)
	}
	if hooksMap == nil {
		return nil
	}

	var eventSlice []any
	if raw, ok := hooksMap["Stop"]; ok {
		eventSlice, _ = raw.([]any)
	}
	if len(eventSlice) == 0 {
		return nil
	}

	// Filter out entries whose command references stop-hook.sh.
	filtered := make([]any, 0, len(eventSlice))
	for _, item := range eventSlice {
		if !hookCommandContains(item, "stop-hook.sh") {
			filtered = append(filtered, item)
		}
	}

	if len(filtered) == 0 {
		delete(hooksMap, "Stop")
	} else {
		hooksMap["Stop"] = filtered
	}

	if len(hooksMap) == 0 {
		delete(existing, "hooks")
	} else {
		existing["hooks"] = hooksMap
	}

	return prov.WriteSettings(existing)
}

// hookCommandContains reports whether a hooks-array entry contains a command
// matching the given substring — used to deduplicate stale entries on re-run.
func hookCommandContains(item any, substr string) bool {
	entry, ok := item.(map[string]any)
	if !ok {
		return false
	}
	hooksRaw, ok := entry["hooks"]
	if !ok {
		return false
	}
	hooks, ok := hooksRaw.([]any)
	if !ok {
		return false
	}
	for _, h := range hooks {
		hMap, ok := h.(map[string]any)
		if !ok {
			continue
		}
		if cmd, ok := hMap["command"].(string); ok && strings.Contains(cmd, substr) {
			return true
		}
	}
	return false
}
