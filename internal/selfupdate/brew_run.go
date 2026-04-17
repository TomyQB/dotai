package selfupdate

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"
)

// RunResult summarises the outcome of a brew update + upgrade sequence.
type RunResult struct {
	// UpdatedDotai is true when brew reported that the dotai formula was
	// reinstalled/upgraded (i.e. the on-disk binary was replaced).
	UpdatedDotai bool

	// ExitErr is the terminal error from the sequence, if any. A failure of
	// `brew update` does not abort the upgrade — only the upgrade result
	// determines success.
	ExitErr error
}

// brewCmd constructs a *exec.Cmd for brew with HOMEBREW_NO_AUTO_UPDATE=1 in
// its environment so it never kicks off a background refresh during the
// user-requested operation. The only invocation that should refresh the
// index is the explicit `brew update` we run ourselves.
func brewCmd(args ...string) *exec.Cmd {
	cmd := exec.Command("brew", args...)
	cmd.Env = brewEnv()
	return cmd
}

// RunBrewUpgrade executes `brew update` followed by `brew upgrade dotai`,
// streaming every stdout/stderr line into onLine as it arrives. The call
// blocks until both commands finish.
//
// onLine is invoked synchronously from a reader goroutine; the caller is
// responsible for making its handler non-blocking (or fast enough to keep up
// with brew output). Passing nil disables streaming.
func RunBrewUpgrade(onLine func(string)) RunResult {
	if onLine != nil {
		onLine("$ brew update")
	}
	// `brew update` explicitly asks for a refresh — bypass the NO_AUTO_UPDATE
	// guard by invoking exec.Command directly so the env does not suppress
	// the very operation we want.
	_ = streamCmd(exec.Command("brew", "update"), onLine)

	if onLine != nil {
		onLine("$ brew upgrade dotai")
	}
	upgradeOut, upgradeErr := streamCmdCollect(brewCmd("upgrade", "dotai"), onLine)

	return RunResult{
		UpdatedDotai: upgradeErr == nil && brewReplacedDotai(upgradeOut),
		ExitErr:      upgradeErr,
	}
}

// RunBrewInstall installs a tap-qualified formula, streaming output line by
// line via onLine. Runs `brew update` first so the tap is fresh — otherwise
// a recent release published under the tap may not be visible locally for
// up to 24 h (HOMEBREW_AUTO_UPDATE_SECS default), and the install would
// fetch a stale bottle version. Returns the exit error from the install
// step (an `update` failure is tolerated and logged but not fatal).
func RunBrewInstall(formula string, onLine func(string)) error {
	if onLine != nil {
		onLine("$ brew update")
	}
	_ = streamCmd(exec.Command("brew", "update"), onLine)

	if onLine != nil {
		onLine("$ brew install " + formula)
	}
	return streamCmd(brewCmd("install", formula), onLine)
}

// RunBrewUninstall uninstalls a formula by bare name (no tap prefix needed),
// streaming output via onLine.
func RunBrewUninstall(formula string, onLine func(string)) error {
	if onLine != nil {
		onLine("$ brew uninstall " + formula)
	}
	return streamCmd(brewCmd("uninstall", formula), onLine)
}

// IsBrewFormulaInstalled reports whether a given formula name is present in
// `brew list --formula`. Bounded by detectTimeout so a hung brew (lock from
// another process, auto-update fetching a slow mirror) cannot stall the UI.
// Any error — timeout, non-zero exit, brew not on PATH — is treated as
// "not installed" rather than surfaced, so callers stay on a simple bool.
func IsBrewFormulaInstalled(name string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), detectTimeout)
	defer cancel()

	out, err := brewListContext(ctx)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == name {
			return true
		}
	}
	return false
}

// brewReplacedDotai inspects the combined stdout+stderr of `brew upgrade dotai`
// and decides whether the formula was actually reinstalled. Brew prints
// "dotai X.Y.Z already installed" (no action) vs "==> Upgrading ... dotai" /
// "==> Pouring dotai..." when it replaces the binary.
func brewReplacedDotai(output string) bool {
	out := strings.ToLower(output)
	upgrading := strings.Contains(out, "==> upgrading") ||
		strings.Contains(out, "==> pouring") ||
		strings.Contains(out, "==> installing dependencies")
	if strings.Contains(out, "already installed") && !upgrading {
		return false
	}
	return upgrading
}

// streamCmd runs cmd, forwarding every line from stdout+stderr to onLine.
// Returns the command's exit error (if any). stdout/stderr are merged into
// one interleaved stream in the order brew emitted them.
func streamCmd(cmd *exec.Cmd, onLine func(string)) error {
	_, err := streamCmdCollect(cmd, onLine)
	return err
}

// streamCmdCollect behaves like streamCmd and additionally returns the full
// combined output so callers can post-process it (e.g. detect "already
// installed" from brew).
func streamCmdCollect(cmd *exec.Cmd, onLine func(string)) (string, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	var (
		mu      sync.Mutex
		all     []byte
		wg      sync.WaitGroup
	)

	drain := func(r io.Reader) {
		defer wg.Done()
		scan := bufio.NewScanner(r)
		// Brew output stays within normal line sizes but bump the buffer to
		// be safe with unusually long dependency lines.
		scan.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scan.Scan() {
			line := scan.Text()
			mu.Lock()
			all = append(all, line...)
			all = append(all, '\n')
			mu.Unlock()
			if onLine != nil {
				onLine(line)
			}
		}
	}

	wg.Add(2)
	go drain(stdout)
	go drain(stderr)
	wg.Wait()

	waitErr := cmd.Wait()
	if waitErr != nil {
		// Surface a terse error that includes brew's exit code for the UI.
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			return string(all), exitErr
		}
		return string(all), waitErr
	}
	return string(all), nil
}
