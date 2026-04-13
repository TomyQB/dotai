// Package provider defines the abstraction layer for AI tool configuration
// directories. Each supported AI tool (Claude Code, etc.) implements Provider
// so the wizard can remain tool-agnostic.
package provider

// Feature represents an optional capability that a provider may or may not
// support. The wizard skips steps whose required feature is not supported.
type Feature int

const (
	// FeatureSettings indicates the provider supports base settings management.
	FeatureSettings Feature = iota
	// FeatureStatusLine indicates the provider supports shell status-line integration.
	FeatureStatusLine
	// FeatureHooks indicates the provider supports event hooks.
	FeatureHooks
	// FeatureProfiles indicates the provider supports profile (CLAUDE.md) management.
	FeatureProfiles
	// FeatureSkills indicates the provider supports skill installation.
	FeatureSkills
)

// Provider is the interface that all AI tool backends must implement.
// It gives the wizard a uniform view of where configuration lives and how to
// read/write it, regardless of which tool is being configured.
type Provider interface {
	// Name returns the human-readable name of the provider (e.g. "Claude Code").
	Name() string

	// ConfigDir returns the absolute path to the provider's configuration
	// directory (e.g. ~/.claude). Creates the directory if it does not exist.
	ConfigDir() (string, error)

	// SettingsFile returns the base filename of the settings file (e.g. "settings.json").
	SettingsFile() string

	// SkillsDir returns the relative path within ConfigDir where skills live.
	SkillsDir() string

	// AgentsDir returns the relative path within ConfigDir where agent files live.
	AgentsDir() string

	// ToolDir returns the relative path within ConfigDir for tool-specific files.
	ToolDir() string

	// ReadSettings reads and parses the provider settings file. Returns an
	// empty map if the file does not exist, and an error for corrupt JSON.
	ReadSettings() (map[string]any, error)

	// WriteSettings serialises data and atomically replaces the settings file.
	WriteSettings(data map[string]any) error

	// SupportsFeature reports whether the provider supports the given feature.
	SupportsFeature(f Feature) bool
}
