package doctor

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// setupFakeHome creates an empty fake $HOME and points the registry env
// override at an empty temp file. Returns the fake home path.
func setupFakeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	regDir := filepath.Join(home, ".claude", "memcli")
	if err := os.MkdirAll(regDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Setenv("MEMCLI_REGISTRY", filepath.Join(regDir, "projects.json"))
	return home
}

// makeGitRepo creates a dir with a .git subdir (so isGitRepo returns true).
func makeGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	return dir
}

func TestInspect_IsPure_NoOutput(t *testing.T) {
	_ = setupFakeHome(t)
	repo := makeGitRepo(t)

	// Capture stdout/stderr to assert Inspect never writes.
	origStdout := os.Stdout
	origStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	_ = Inspect(repo)

	_ = wOut.Close()
	_ = wErr.Close()
	var bufOut, bufErr bytes.Buffer
	_, _ = io.Copy(&bufOut, rOut)
	_, _ = io.Copy(&bufErr, rErr)
	if bufOut.Len() != 0 {
		t.Errorf("Inspect wrote to stdout: %q", bufOut.String())
	}
	if bufErr.Len() != 0 {
		t.Errorf("Inspect wrote to stderr: %q", bufErr.String())
	}
}

func TestInspect_FailingFixture_HasFailures(t *testing.T) {
	_ = setupFakeHome(t)
	repo := makeGitRepo(t)
	r := Inspect(repo)
	if !r.HasFailures() {
		t.Error("empty fake-home fixture should have failures")
	}
	if len(r.Global) == 0 {
		t.Error("expected Global checks populated")
	}
	if len(r.Project) == 0 {
		t.Error("expected Project checks populated (repo is git)")
	}
}

func TestInspect_NonGitRepo_NoProjectChecks(t *testing.T) {
	_ = setupFakeHome(t)
	dir := t.TempDir() // no .git inside
	r := Inspect(dir)
	if len(r.Project) != 0 {
		t.Errorf("non-git repo should have no Project checks, got %d", len(r.Project))
	}
}

// TestReport_String_Golden locks the exact byte layout of Report.String()
// against a known, static Report. This is the byte-for-byte CLI output
// contract from spec REQ-8.
func TestReport_String_Golden(t *testing.T) {
	report := Report{
		RepoPath: "/fake/repo",
		Global: []Check{
			{Name: "skill memcli-init", Status: StatusOK},
			{Name: "skill memcli-update", Status: StatusFail, Detail: "missing: /tmp/x  (run: memcli install)"},
		},
		Project: []Check{
			{Name: "project .agent-memory/index.md", Status: StatusOK},
			{Name: ".stale status", Status: StatusOK},
		},
	}
	goldenPath := filepath.Join("testdata", "report_golden.txt")
	got := report.String()
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) && os.Getenv("UPDATE_GOLDEN") == "1" {
			if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
			return
		}
		t.Fatalf("read golden: %v", err)
	}
	if got != string(want) {
		t.Errorf("Report.String() mismatch.\n--- got ---\n%s\n--- want ---\n%s", got, string(want))
	}
}

func TestReport_HasFailures(t *testing.T) {
	ok := Report{Global: []Check{{Name: "x", Status: StatusOK}}}
	if ok.HasFailures() {
		t.Error("OK report should not have failures")
	}
	bad := Report{Project: []Check{{Name: "x", Status: StatusFail}}}
	if !bad.HasFailures() {
		t.Error("report with failure should report HasFailures=true")
	}
}
