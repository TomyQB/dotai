//go:build !windows

package selfupdate

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// RelaunchSelfWith uses execve(2) to replace the current process with a fresh
// invocation of the dotai binary, passing extraArgs and inheriting the current
// environment. On success it never returns — the caller's process image is
// replaced.
//
// Used after `brew upgrade dotai` reinstalls the on-disk binary so the update
// screen can continue executing with the new code path (and the new embedded
// asset set).
//
// Resolution strategy: we cannot use /proc/self/exe (Linux) because that
// symlink points at the old inode that brew just replaced. We cannot blindly
// reuse os.Args[0] either (it may be a bare name). Instead we take the
// basename of argv[0] and let exec.LookPath walk $PATH — which, after brew
// upgrade, resolves through /opt/homebrew/bin/dotai (a symlink) to the newly
// installed binary.
func RelaunchSelfWith(extraArgs ...string) error {
	name := filepath.Base(os.Args[0])
	exe, err := exec.LookPath(name)
	if err != nil {
		return err
	}
	argv := append([]string{exe}, extraArgs...)
	return syscall.Exec(exe, argv, os.Environ())
}
