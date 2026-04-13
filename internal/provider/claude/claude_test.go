package claude_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TomyQB/dotai/internal/provider/claude"
)

// newProviderWithDir constructs a Claude Provider but overrides its config
// directory to dir so tests never touch ~/.claude.
// We do this by setting HOME to a temp dir that has a .claude subdirectory.
func newProviderWithHome(t *testing.T) (p *claude.Provider, claudeDir string) {
	t.Helper()
	home := t.TempDir()
	claudeDir = filepath.Join(home, ".claude")
	t.Setenv("HOME", home)
	p = claude.New()
	return p, claudeDir
}

// ---------------------------------------------------------------------------
// ConfigDir
// ---------------------------------------------------------------------------

func TestConfigDir_CreatesDirectory(t *testing.T) {
	p, want := newProviderWithHome(t)

	got, err := p.ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error: %v", err)
	}
	if got != want {
		t.Errorf("ConfigDir() = %q, want %q", got, want)
	}
	if _, err := os.Stat(got); err != nil {
		t.Errorf("ConfigDir() did not create directory: %v", err)
	}
}

func TestConfigDir_Caching(t *testing.T) {
	p, _ := newProviderWithHome(t)

	first, err := p.ConfigDir()
	if err != nil {
		t.Fatalf("first ConfigDir(): %v", err)
	}
	second, err := p.ConfigDir()
	if err != nil {
		t.Fatalf("second ConfigDir(): %v", err)
	}
	if first != second {
		t.Errorf("ConfigDir() returned different values on second call: %q vs %q", first, second)
	}
}

// ---------------------------------------------------------------------------
// ReadSettings
// ---------------------------------------------------------------------------

func TestReadSettings(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(dir string) // write files before the call
		wantMap map[string]any
		wantErr bool
	}{
		{
			name:    "missing file returns empty map",
			setup:   func(_ string) {},
			wantMap: map[string]any{},
		},
		{
			name: "valid JSON returns map",
			setup: func(dir string) {
				data := `{"foo":"bar","num":42}` + "\n"
				os.WriteFile(filepath.Join(dir, "settings.json"), []byte(data), 0o644)
			},
			wantMap: map[string]any{"foo": "bar", "num": float64(42)},
		},
		{
			name: "corrupt JSON returns error",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{bad json`), 0o644)
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, claudeDir := newProviderWithHome(t)
			os.MkdirAll(claudeDir, 0o755)
			tc.setup(claudeDir)

			got, err := p.ReadSettings()
			if (err != nil) != tc.wantErr {
				t.Fatalf("ReadSettings() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			for k, want := range tc.wantMap {
				if got[k] != want {
					t.Errorf("ReadSettings()[%q] = %v, want %v", k, got[k], want)
				}
			}
			if len(got) != len(tc.wantMap) {
				t.Errorf("ReadSettings() returned %d keys, want %d", len(got), len(tc.wantMap))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// WriteSettings
// ---------------------------------------------------------------------------

func TestWriteSettings(t *testing.T) {
	t.Run("creates file with indented JSON and trailing newline", func(t *testing.T) {
		p, claudeDir := newProviderWithHome(t)
		os.MkdirAll(claudeDir, 0o755)

		data := map[string]any{"key": "value", "num": float64(1)}
		if err := p.WriteSettings(data); err != nil {
			t.Fatalf("WriteSettings() error: %v", err)
		}

		raw, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		// Must end with newline.
		if !strings.HasSuffix(string(raw), "\n") {
			t.Error("WriteSettings() output does not end with newline")
		}
		// Must be valid JSON.
		var out map[string]any
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}
		// Must be indented (contains two-space indent).
		if !strings.Contains(string(raw), "  ") {
			t.Error("WriteSettings() output is not indented")
		}
	})

	t.Run("round-trip: write then read returns same data", func(t *testing.T) {
		p, claudeDir := newProviderWithHome(t)
		os.MkdirAll(claudeDir, 0o755)

		in := map[string]any{"hello": "world"}
		if err := p.WriteSettings(in); err != nil {
			t.Fatalf("WriteSettings: %v", err)
		}
		out, err := p.ReadSettings()
		if err != nil {
			t.Fatalf("ReadSettings: %v", err)
		}
		if out["hello"] != "world" {
			t.Errorf("round-trip failed: got %v", out)
		}
	})

	t.Run("atomic write: no partial file on second write", func(t *testing.T) {
		p, claudeDir := newProviderWithHome(t)
		os.MkdirAll(claudeDir, 0o755)

		first := map[string]any{"v": "1"}
		if err := p.WriteSettings(first); err != nil {
			t.Fatalf("first write: %v", err)
		}
		second := map[string]any{"v": "2"}
		if err := p.WriteSettings(second); err != nil {
			t.Fatalf("second write: %v", err)
		}
		out, _ := p.ReadSettings()
		if out["v"] != "2" {
			t.Errorf("expected v=2 after second write, got %v", out["v"])
		}
		// No stray temp files.
		entries, _ := os.ReadDir(claudeDir)
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".settings-") {
				t.Errorf("stray temp file left: %s", e.Name())
			}
		}
	})
}
