package memclitui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/memclitui"
	"github.com/TomyQB/dotai/internal/registry"
)

// setupProject builds a temp project with a nested .agent-memory/ tree so the
// browser has something to walk:
//
//	<tmp>/proj/.agent-memory/
//	├── flows/
//	│   └── api/
//	│       └── payments.md
//	└── root.md
func setupProject(t *testing.T) registry.Entry {
	t.Helper()
	root := filepath.Join(t.TempDir(), "proj")
	base := filepath.Join(root, ".agent-memory")
	if err := os.MkdirAll(filepath.Join(base, "flows", "api"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	files := map[string]string{
		filepath.Join(base, "root.md"):                       "# root",
		filepath.Join(base, "flows", "api", "payments.md"):   "# payments",
	}
	for p, body := range files {
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}
	return registry.Entry{Path: root, Name: "proj"}
}

// TestBrowserDrillDown walks the user interaction: enter flows/ → enter api/
// → confirm the cursor sees payments.md.
func TestBrowserDrillDown(t *testing.T) {
	e := setupProject(t)
	m := memclitui.NewBrowser(e)

	// Root: flows/ appears first (dirs before files), cursor defaults to 0.
	if got := m.Title(); got != "proj" {
		t.Errorf("Title at root = %q, want %q", got, "proj")
	}

	// Enter flows/
	after, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = after.(memclitui.BrowserModel)
	if got := m.Title(); got != "proj / flows" {
		t.Errorf("Title after entering flows = %q, want %q", got, "proj / flows")
	}

	// Enter api/
	after, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = after.(memclitui.BrowserModel)
	if got := m.Title(); got != "proj / flows/api" {
		t.Errorf("Title after entering api = %q, want %q", got, "proj / flows/api")
	}
}

// TestBrowserLeftGoesUpOneLevel confirms that ← climbs one breadcrumb at a
// time, not all the way out of the project.
func TestBrowserLeftGoesUpOneLevel(t *testing.T) {
	e := setupProject(t)
	m := memclitui.NewBrowser(e)

	// Drill into flows/ then api/
	after, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = after.(memclitui.BrowserModel)
	after, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = after.(memclitui.BrowserModel)

	// ← climbs one level: api → flows
	after, cmd := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = after.(memclitui.BrowserModel)
	if cmd != nil {
		// Inside the tree, left must NOT pop the screen.
		if _, isPop := cmd().(memclitui.PopScreenMsg); isPop {
			t.Error("left at depth 2 emitted PopScreenMsg, want in-place climb")
		}
	}
	if got := m.Title(); got != "proj / flows" {
		t.Errorf("Title after left = %q, want %q", got, "proj / flows")
	}

	// Second ← climbs to root.
	after, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = after.(memclitui.BrowserModel)
	if got := m.Title(); got != "proj" {
		t.Errorf("Title after second left = %q, want %q", got, "proj")
	}
}

// TestBrowserLeftAtRootPops verifies that ← at the project root emits
// PopScreenMsg so the user returns to the projects list, instead of being
// stuck inside the browser.
func TestBrowserLeftAtRootPops(t *testing.T) {
	e := setupProject(t)
	m := memclitui.NewBrowser(e)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if cmd == nil {
		t.Fatal("left at root returned nil cmd, want PopScreenMsg")
	}
	if _, ok := cmd().(memclitui.PopScreenMsg); !ok {
		t.Errorf("left at root emitted %T, want memclitui.PopScreenMsg", cmd())
	}
}

// TestBrowserEscAlwaysPops confirms that esc pops the whole browser
// regardless of depth (matches the convention used by every other screen in
// the dotai ecosystem).
func TestBrowserEscAlwaysPops(t *testing.T) {
	e := setupProject(t)
	m := memclitui.NewBrowser(e)

	// Drill into flows/
	after, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = after.(memclitui.BrowserModel)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc returned nil cmd, want PopScreenMsg")
	}
	if _, ok := cmd().(memclitui.PopScreenMsg); !ok {
		t.Errorf("esc emitted %T, want memclitui.PopScreenMsg", cmd())
	}
}

// TestPushDeliversWindowSize guards the v0.3.6 regression where a screen
// pushed mid-session came up with zero dimensions (the Preview model sized
// its viewport as 10x3 and showed nothing useful). Pushing a preview after
// a WindowSizeMsg has already configured the root must leave the top
// screen's View output non-empty.
func TestPushDeliversWindowSize(t *testing.T) {
	e := setupProject(t)
	mdPath := filepath.Join(e.Path, ".agent-memory", "root.md")

	app := memclitui.New(&registry.Registry{Version: registry.SchemaVersion})
	// Prime the root with a terminal size, mimicking Bubbletea's initial msg.
	appModel, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = appModel.(memclitui.Model)

	// Push a preview screen — the root must propagate the size so the
	// viewport inside Preview gets configured on first render.
	appModel, _ = app.Update(memclitui.PushScreenMsg{Screen: memclitui.NewPreview(mdPath)})
	app = appModel.(memclitui.Model)

	view := app.View()
	if view == "" {
		t.Fatal("View() after push returned empty string — size was not delivered")
	}
	// The body is loaded at NewPreview, so the file content must appear in
	// the rendered viewport once width/height are > 0.
	if !strings.Contains(view, "# root") {
		t.Errorf("View() does not contain the preview body; size delivery regressed")
	}
}
