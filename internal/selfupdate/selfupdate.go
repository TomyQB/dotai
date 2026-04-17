// Package selfupdate encapsulates upgrading the dotai binary itself.
//
// Today the only supported channel is Homebrew on macOS/Linux. The package
// exposes a detection layer (is brew on PATH? is dotai managed by brew?) and
// a runner that streams `brew update && brew upgrade dotai` output line by
// line so the UI can show live progress. If brew is absent or dotai is not
// managed by it, callers are expected to skip the upgrade step and proceed
// with config-file re-apply only.
package selfupdate

import (
	"context"
	"os"
	"os/exec"
	"time"
)

// detectTimeout caps how long the detection phase waits for brew to answer.
// `brew list` is normally instantaneous (~15 ms), but with
// HOMEBREW_AUTO_UPDATE it can trigger a background `brew update` that takes
// minutes, and if another brew is running the list blocks on the tap lock.
// Either way the UI must not hang — we fall back to "not managed" after
// detectTimeout and let the user retry. Kept tight (3 s) because the
// calling screen shows a spinner during the wait.
const detectTimeout = 3 * time.Second

// Availability describes whether a brew-based self-update is viable on this
// machine right now. All fields are false on Windows and on Linux systems
// without Homebrew/Linuxbrew installed.
type Availability struct {
	// BrewOnPath reports whether the `brew` binary is reachable.
	BrewOnPath bool

	// DotaiManaged reports whether brew lists dotai as an installed formula.
	// Meaningful only when BrewOnPath is true.
	DotaiManaged bool
}

// CanUpgrade is the single predicate callers use to decide whether to run
// the brew phase. Both conditions must hold.
func (a Availability) CanUpgrade() bool {
	return a.BrewOnPath && a.DotaiManaged
}

// Detect inspects the environment and returns the current Availability. It
// is bounded by detectTimeout so a hung brew cannot freeze the caller.
func Detect() Availability {
	a := Availability{}
	if _, err := exec.LookPath("brew"); err != nil {
		return a
	}
	a.BrewOnPath = true
	a.DotaiManaged = IsBrewFormulaInstalled("dotai")
	return a
}

// brewEnv returns the environment brew should run in during detection and
// long-running commands alike. HOMEBREW_NO_AUTO_UPDATE=1 stops brew from
// sneakily running `brew update` in the background before the user's
// requested operation, which otherwise turns a 15ms list into a multi-minute
// network fetch. The explicit `brew update` we run in RunBrewUpgrade is not
// affected — that invocation asks for the refresh directly.
func brewEnv() []string {
	env := os.Environ()
	env = append(env, "HOMEBREW_NO_AUTO_UPDATE=1")
	return env
}

// brewListContext runs `brew list --formula -1` bound by ctx. Returns the
// raw newline-delimited output, or an empty slice + error on timeout / exit.
func brewListContext(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "brew", "list", "--formula", "-1")
	cmd.Env = brewEnv()
	return cmd.Output()
}
