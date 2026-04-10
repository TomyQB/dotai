// Package messages defines the shared message types and helper commands used
// across TUI screens. Extracting these into a leaf package breaks import
// cycles between the root model and individual screen packages.
package messages

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/mem-cli/internal/doctor"
	"github.com/TomyQB/mem-cli/internal/registry"
)

// Screen is the contract every TUI screen implements.
type Screen interface {
	tea.Model
	Title() string
}

// ProjectSelectedMsg is emitted by Home when the user opens a project.
type ProjectSelectedMsg struct {
	Entry registry.Entry
}

// DocSelectedMsg is emitted by Project when the user opens a .md file.
type DocSelectedMsg struct {
	Path string
}

// DoctorResultMsg carries a successful doctor.Inspect result.
type DoctorResultMsg struct {
	Report doctor.Report
}

// DoctorErrMsg carries an error (including recovered panic) from doctor run.
type DoctorErrMsg struct {
	Err error
}

// RefreshMsg triggers a re-read of the registry on Home.
type RefreshMsg struct{}

// RegistryLoadedMsg carries the result of an async Load().
type RegistryLoadedMsg struct {
	Reg *registry.Registry
	Err error
}

// PopScreenMsg asks the root to pop the top screen from the stack.
type PopScreenMsg struct{}

// DismissModalMsg asks the root to clear any open modal.
type DismissModalMsg struct{}

// EmptyReport returns a zero-value Report (used when doctor errors out
// before producing a result — keeps the modal renderer simple).
func EmptyReport() doctor.Report {
	return doctor.Report{}
}

// RunDoctorCmd returns a tea.Cmd that executes doctor.Inspect asynchronously.
// Any panic is recovered and surfaced as DoctorErrMsg so the TUI stays alive.
func RunDoctorCmd(path string) tea.Cmd {
	return func() (msg tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				msg = DoctorErrMsg{Err: recoveredError(r)}
			}
		}()
		return DoctorResultMsg{Report: doctor.Inspect(path)}
	}
}

// LoadRegistryCmd returns a tea.Cmd that reloads the registry from disk.
func LoadRegistryCmd() tea.Cmd {
	return func() tea.Msg {
		reg, err := registry.Load()
		return RegistryLoadedMsg{Reg: reg, Err: err}
	}
}

// recoveredError converts the result of recover() into an error.
func recoveredError(r any) error {
	if err, ok := r.(error); ok {
		return err
	}
	return &panicError{v: r}
}

type panicError struct{ v any }

func (e *panicError) Error() string {
	return "panic in doctor.Inspect: " + toString(e.v)
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case error:
		return x.Error()
	default:
		return "unknown"
	}
}
