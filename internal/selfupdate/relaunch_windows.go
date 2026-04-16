//go:build windows

package selfupdate

import "errors"

// RelaunchSelfWith is a no-op on Windows: brew is not supported there, so the
// upstream flow never reaches this code path in practice. We keep the symbol
// so the package compiles on every GOOS.
func RelaunchSelfWith(_ ...string) error {
	return errors.New("self-relaunch is unsupported on windows")
}
