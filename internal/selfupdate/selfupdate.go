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
	"os/exec"
	"strings"
)

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

// Detect inspects the environment and returns the current Availability.
func Detect() Availability {
	a := Availability{}
	if _, err := exec.LookPath("brew"); err != nil {
		return a
	}
	a.BrewOnPath = true
	a.DotaiManaged = isDotaiBrewFormula()
	return a
}

// isDotaiBrewFormula runs `brew list --formula` and scans for dotai. Using
// --formula avoids colliding with any cask/tap of the same name and keeps
// the match exact. Any non-zero exit from brew is treated as "not managed"
// rather than propagating — the caller simply skips the upgrade phase.
func isDotaiBrewFormula() bool {
	out, err := exec.Command("brew", "list", "--formula", "-1").Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == "dotai" {
			return true
		}
	}
	return false
}
