package registry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// withRegistryEnv points MEMCLI_REGISTRY at a temp file and returns its path.
// Automatically unsets the env var on test cleanup.
func withRegistryEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "projects.json")
	t.Setenv(envOverride, path)
	return path
}

func copyFixture(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read fixture %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write fixture to %s: %v", dst, err)
	}
}

func TestPath_EnvOverride(t *testing.T) {
	t.Setenv(envOverride, "/custom/reg.json")
	if got := Path(); got != "/custom/reg.json" {
		t.Fatalf("Path() = %q, want /custom/reg.json", got)
	}
}

func TestPath_Default(t *testing.T) {
	t.Setenv(envOverride, "")
	got := Path()
	if filepath.Base(got) != "projects.json" {
		t.Fatalf("Path() base = %q, want projects.json", filepath.Base(got))
	}
	if filepath.Base(filepath.Dir(got)) != "memcli" {
		t.Fatalf("Path() parent = %q, want memcli", filepath.Base(filepath.Dir(got)))
	}
}

func TestLoad_Missing(t *testing.T) {
	_ = withRegistryEnv(t)
	r, err := Load()
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if r.Version != SchemaVersion {
		t.Errorf("version = %d, want %d", r.Version, SchemaVersion)
	}
	if len(r.Projects) != 0 {
		t.Errorf("projects = %d, want 0", len(r.Projects))
	}
}

func TestLoad_V1Migrates(t *testing.T) {
	path := withRegistryEnv(t)
	copyFixture(t, "testdata/registry_v1.json", path)
	r, err := Load()
	if err != nil {
		t.Fatalf("Load v1: %v", err)
	}
	if len(r.Projects) != 2 {
		t.Fatalf("projects = %d, want 2", len(r.Projects))
	}
	if r.Projects[0].Path != "/tmp/memcli-fixture-a" {
		t.Errorf("projects[0].Path = %q", r.Projects[0].Path)
	}
	if r.Projects[0].Name != "memcli-fixture-a" {
		t.Errorf("projects[0].Name = %q", r.Projects[0].Name)
	}
	if !r.Projects[0].InitializedAt.IsZero() {
		t.Errorf("projects[0].InitializedAt = %v, want zero", r.Projects[0].InitializedAt)
	}
}

func TestLoad_V2RoundTrip(t *testing.T) {
	path := withRegistryEnv(t)
	copyFixture(t, "testdata/registry_v2.json", path)
	r, err := Load()
	if err != nil {
		t.Fatalf("Load v2: %v", err)
	}
	if len(r.Projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(r.Projects))
	}
	want := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	if !r.Projects[0].InitializedAt.Equal(want) {
		t.Errorf("InitializedAt = %v, want %v", r.Projects[0].InitializedAt, want)
	}
}

func TestLoad_UnknownFields(t *testing.T) {
	path := withRegistryEnv(t)
	copyFixture(t, "testdata/registry_unknown_field.json", path)
	r, err := Load()
	if err != nil {
		t.Fatalf("Load unknown field: %v", err)
	}
	if len(r.Projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(r.Projects))
	}
}

func TestLoad_Malformed(t *testing.T) {
	path := withRegistryEnv(t)
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := Load()
	if err == nil {
		t.Fatal("want error on malformed JSON, got nil")
	}
	var regErr *RegistryError
	if !errors.As(err, &regErr) {
		t.Errorf("want *RegistryError, got %T", err)
	}
}

func TestSave_WritesV2AfterV1Read(t *testing.T) {
	path := withRegistryEnv(t)
	copyFixture(t, "testdata/registry_v1.json", path)
	r, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := r.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var check struct {
		Version  int `json:"version"`
		Projects []struct {
			Path string `json:"path"`
			Name string `json:"name"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(data, &check); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if check.Version != 2 {
		t.Errorf("version on disk = %d, want 2", check.Version)
	}
	if len(check.Projects) != 2 {
		t.Fatalf("projects on disk = %d, want 2", len(check.Projects))
	}
	if check.Projects[0].Name != "memcli-fixture-a" {
		t.Errorf("name[0] = %q", check.Projects[0].Name)
	}
}

func TestPrune(t *testing.T) {
	path := withRegistryEnv(t)
	_ = path
	existing := t.TempDir()
	r := &Registry{
		Version: 2,
		Projects: []Entry{
			{Path: existing, Name: filepath.Base(existing)},
			{Path: "/nonexistent/path/xyz-123", Name: "xyz"},
		},
	}
	removed := r.Prune()
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
	if len(r.Projects) != 1 {
		t.Errorf("remaining = %d, want 1", len(r.Projects))
	}
	if r.Projects[0].Path != existing {
		t.Errorf("wrong entry kept: %q", r.Projects[0].Path)
	}
}

func TestTouch_NewAndUpdate(t *testing.T) {
	r := &Registry{Version: 2}
	r.Touch("/tmp/touch-a")
	if len(r.Projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(r.Projects))
	}
	if r.Projects[0].InitializedAt.IsZero() {
		t.Error("InitializedAt should be set on new entry")
	}
	if r.Projects[0].LastSeen.IsZero() {
		t.Error("LastSeen should be set on new entry")
	}
	initAt := r.Projects[0].InitializedAt
	firstSeen := r.Projects[0].LastSeen
	time.Sleep(2 * time.Millisecond)

	r.Touch("/tmp/touch-a")
	if len(r.Projects) != 1 {
		t.Errorf("projects after second touch = %d, want 1", len(r.Projects))
	}
	if !r.Projects[0].InitializedAt.Equal(initAt) {
		t.Errorf("InitializedAt changed on second touch")
	}
	if !r.Projects[0].LastSeen.After(firstSeen) {
		t.Errorf("LastSeen did not advance: first=%v now=%v", firstSeen, r.Projects[0].LastSeen)
	}
}

func TestFind(t *testing.T) {
	r := &Registry{
		Projects: []Entry{{Path: "/a"}, {Path: "/b"}},
	}
	if _, ok := r.Find("/a"); !ok {
		t.Error("Find(/a) should succeed")
	}
	if _, ok := r.Find("/missing"); ok {
		t.Error("Find(/missing) should fail")
	}
}
