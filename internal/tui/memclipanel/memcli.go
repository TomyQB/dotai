// Package memclipanel implements the memcli management screen for dotai.
//
// The screen offers four actions:
//
//	Install binary        brew install TomyQB/tap/memcli  (streams output)
//	Install integration   skills + agent + Stop hook (via wizard.InstallMemcli)
//	Uninstall integration wizard.UninstallMemcli
//	Back                  return to the main menu
//
// Binary install is brew-only: if brew is not on $PATH the action surfaces a
// toast pointing at the GitHub release. All long-running commands stream
// stdout/stderr into a tail visible in the UI while they run.
package memclipanel

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/detect"
	"github.com/TomyQB/dotai/internal/provider"
	"github.com/TomyQB/dotai/internal/selfupdate"
	"github.com/TomyQB/dotai/internal/tui/banner"
	"github.com/TomyQB/dotai/internal/tui/messages"
	"github.com/TomyQB/dotai/internal/tui/styles"
	"github.com/TomyQB/dotai/internal/wizard"
)

// phase tracks what the screen is currently doing.
type phase int

const (
	phaseIdle        phase = iota // menu is interactive
	phaseBrewRun                  // brew install/uninstall running — streaming
	phaseIntegration              // wizard integration install/uninstall running
)

// brewTailLen is how many trailing brew lines we keep on screen.
const brewTailLen = 6

// memcliFormula is the full tap-qualified formula for the first install.
// Subsequent calls that only need the bare name (uninstall, list) use "memcli".
const memcliFormula = "TomyQB/tap/memcli"

// memcliReleaseURL is shown to the user when brew is not available so they
// can install the binary manually.
const memcliReleaseURL = "https://github.com/TomyQB/dotai/releases"

// statusMsg carries the result of the initial status check.
type statusMsg struct {
	integrationInstalled bool
	binaryInstalled      bool
	brewAvailable        bool
}

// brewLineMsg delivers a single stdout/stderr line from the running brew
// sub-process so the UI can append it to the live tail.
type brewLineMsg struct{ line string }

// brewDoneMsg signals the brew command has finished.
type brewDoneMsg struct {
	action string
	err    error
}

// integrationDoneMsg signals the wizard integration install/uninstall ended.
type integrationDoneMsg struct {
	action string
	err    error
}

// Model is the Bubble Tea model for the memcli management screen.
type Model struct {
	prov                 provider.Provider
	phase                phase
	loaded               bool
	integrationInstalled bool
	binaryInstalled      bool
	brewAvailable        bool
	brewTail             []string
	brewWait             tea.Cmd
	cursor               int
	toast                string
	toastErr             bool
	width, height        int
}

// New returns an initialised memclipanel Model.
func New(prov provider.Provider) Model {
	return Model{prov: prov}
}

// Title satisfies messages.Screen.
func (m Model) Title() string { return "memcli" }

// Init kicks off the status check that populates the screen's flags.
func (m Model) Init() tea.Cmd {
	prov := m.prov
	return func() tea.Msg {
		avail := selfupdate.Detect()
		integ := detect.CheckComponent(prov, detect.Memcli).Installed
		bin := false
		if avail.BrewOnPath {
			bin = selfupdate.IsBrewFormulaInstalled("memcli")
		}
		return statusMsg{
			integrationInstalled: integ,
			binaryInstalled:      bin,
			brewAvailable:        avail.BrewOnPath,
		}
	}
}

// Update handles incoming messages and key events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statusMsg:
		m.integrationInstalled = msg.integrationInstalled
		m.binaryInstalled = msg.binaryInstalled
		m.brewAvailable = msg.brewAvailable
		m.loaded = true
		return m, nil

	case brewLineMsg:
		m.brewTail = appendTail(m.brewTail, msg.line, brewTailLen)
		return m, m.brewWait

	case brewDoneMsg:
		m.phase = phaseIdle
		m.brewWait = nil
		if msg.err != nil {
			m.toast = msg.action + " failed: " + msg.err.Error()
			m.toastErr = true
		} else {
			m.toast = msg.action + " completed"
			m.toastErr = false
		}
		return m, m.Init()

	case integrationDoneMsg:
		m.phase = phaseIdle
		if msg.err != nil {
			m.toast = msg.action + " failed: " + msg.err.Error()
			m.toastErr = true
		} else {
			m.toast = msg.action + " completed"
			m.toastErr = false
		}
		return m, m.Init()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// handleKey routes keyboard input based on the current phase.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.phase != phaseIdle {
		return m, nil
	}

	actions := m.actions()

	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(actions)-1 {
			m.cursor++
		}
	case "enter", "right":
		return m.runSelected(actions)
	case "esc", "left", "q":
		return m, popCmd()
	}
	return m, nil
}

// runSelected dispatches the action at the current cursor position.
func (m Model) runSelected(actions []action) (tea.Model, tea.Cmd) {
	if m.cursor >= len(actions) {
		return m, nil
	}
	a := actions[m.cursor]
	if a.disabled {
		return m, nil
	}
	m.toast = ""
	m.toastErr = false
	switch a.id {
	case actInstallBinary:
		return m.startBrewInstallBinary()
	case actUninstallBinary:
		return m.startBrewUninstallBinary()
	case actInstallIntegration:
		m.phase = phaseIntegration
		return m, runIntegrationCmd(m.prov, true)
	case actUninstallIntegration:
		m.phase = phaseIntegration
		return m, runIntegrationCmd(m.prov, false)
	case actBack:
		return m, popCmd()
	}
	return m, nil
}

// startBrewInstallBinary begins the brew install sequence. If brew is not on
// PATH it short-circuits with an informative toast and stays in phaseIdle.
func (m Model) startBrewInstallBinary() (tea.Model, tea.Cmd) {
	if !m.brewAvailable {
		m.toast = "brew not found on PATH — install memcli manually from " + memcliReleaseURL
		m.toastErr = true
		return m, nil
	}
	m.phase = phaseBrewRun
	m.brewTail = nil
	runner, waiter := brewCmds("install binary", func(onLine func(string)) error {
		return selfupdate.RunBrewInstall(memcliFormula, onLine)
	})
	m.brewWait = waiter
	return m, tea.Batch(runner, waiter)
}

// startBrewUninstallBinary triggers `brew uninstall memcli` with streaming UI.
func (m Model) startBrewUninstallBinary() (tea.Model, tea.Cmd) {
	if !m.brewAvailable {
		m.toast = "brew not found on PATH — remove memcli manually"
		m.toastErr = true
		return m, nil
	}
	m.phase = phaseBrewRun
	m.brewTail = nil
	runner, waiter := brewCmds("uninstall binary", func(onLine func(string)) error {
		return selfupdate.RunBrewUninstall("memcli", onLine)
	})
	m.brewWait = waiter
	return m, tea.Batch(runner, waiter)
}

// SetSize stores the terminal dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// View renders the memcli panel inside a FrameCompact border.
func (m Model) View() string {
	innerWidth := styles.CompactInnerWidth
	header := banner.RenderCompact(innerWidth, "memcli")

	var body strings.Builder

	if !m.loaded {
		body.WriteString(styles.Help.Render("Checking status..."))
		return styles.FrameCompact(header, body.String(), m.Footer(), styles.CompactWidth)
	}

	body.WriteString(renderStatus(m.binaryInstalled, m.integrationInstalled, m.brewAvailable))
	body.WriteString("\n")

	if m.phase == phaseBrewRun {
		body.WriteString("\n")
		body.WriteString(styles.WarningText.Render("Running brew..."))
		body.WriteString("\n\n")
		body.WriteString(renderTail(m.brewTail))
	} else if m.phase == phaseIntegration {
		body.WriteString("\n")
		body.WriteString(styles.WarningText.Render("Applying integration..."))
	} else {
		body.WriteString("\n")
		body.WriteString(renderActions(m.actions(), m.cursor))
	}

	if m.toast != "" {
		body.WriteString("\n\n")
		if m.toastErr {
			body.WriteString(styles.ProgressError.Render(m.toast))
		} else {
			body.WriteString(styles.Toast.Render(m.toast))
		}
	}

	return styles.FrameCompact(header, body.String(), m.Footer(), styles.CompactWidth)
}

// Footer returns the key-hint line for this screen.
func (m Model) Footer() string {
	if m.phase != phaseIdle {
		return styles.FooterHints("please wait...", "")
	}
	return styles.FooterHints("↑/↓", "navigate", "enter/→", "select", "esc/←", "back")
}

// actionID is the enum of possible memcli management operations.
type actionID int

const (
	actInstallBinary actionID = iota
	actUninstallBinary
	actInstallIntegration
	actUninstallIntegration
	actBack
)

// action pairs an id with its display label and enabled flag.
type action struct {
	id       actionID
	label    string
	disabled bool
}

// actions builds the menu rows based on current installation state. The label
// of idempotent actions switches between Install/Reinstall; uninstall rows
// become disabled when the corresponding piece is not installed.
func (m Model) actions() []action {
	binLabel := "Install binary"
	if m.binaryInstalled {
		binLabel = "Reinstall binary"
	}
	if !m.brewAvailable {
		binLabel += " (brew required)"
	}

	intLabel := "Install integration"
	if m.integrationInstalled {
		intLabel = "Reinstall integration"
	}

	return []action{
		{id: actInstallBinary, label: binLabel},
		{id: actUninstallBinary, label: "Uninstall binary", disabled: !m.binaryInstalled || !m.brewAvailable},
		{id: actInstallIntegration, label: intLabel},
		{id: actUninstallIntegration, label: "Uninstall integration", disabled: !m.integrationInstalled},
		{id: actBack, label: "Back"},
	}
}

// renderStatus prints the two status rows (binary, integration) plus a hint
// when brew is unavailable.
func renderStatus(binInstalled, integInstalled, brewAvail bool) string {
	var b strings.Builder
	b.WriteString(styleStatus("Binary", binInstalled))
	b.WriteString("\n")
	b.WriteString(styleStatus("Integration", integInstalled))
	b.WriteString("\n")
	if !brewAvail {
		b.WriteString(styles.Help.Render("brew not found — binary install/uninstall disabled"))
	}
	return b.String()
}

// styleStatus renders a "Label: ✓ Installed" or "Label: ✗ Not installed" row.
func styleStatus(label string, installed bool) string {
	icon := styles.CheckOff.Render("✗")
	text := styles.Help.Render("Not installed")
	if installed {
		icon = styles.CheckOn.Render("✓")
		text = styles.MenuItem.Render("Installed")
	}
	return label + ": " + icon + " " + text
}

// renderActions paints the action menu with cursor + disabled chrome.
func renderActions(actions []action, cursor int) string {
	var b strings.Builder
	for i, a := range actions {
		prefix := "  "
		var label string
		switch {
		case a.disabled:
			label = styles.CheckOff.Render(a.label)
		case i == cursor:
			prefix = styles.Arrow.Render("▸ ")
			label = styles.MenuItemSelected.Render(a.label)
		default:
			label = styles.MenuItem.Render(a.label)
		}
		b.WriteString(prefix + label)
		if i < len(actions)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderTail prints the last few brew output lines so the user sees progress.
func renderTail(lines []string) string {
	if len(lines) == 0 {
		return styles.Help.Render("  (waiting for brew output...)")
	}
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(styles.Help.Render("  " + line))
		b.WriteString("\n")
	}
	return b.String()
}

// brewCmds returns a matched (runner, waiter) pair for one brew invocation.
// run is the function that actually drives brew with its line callback.
// When run returns, the runner closes the channel and emits brewDoneMsg so
// the waiter terminates gracefully on its next receive.
func brewCmds(action string, run func(onLine func(string)) error) (tea.Cmd, tea.Cmd) {
	ch := make(chan string, 128)
	runner := func() tea.Msg {
		err := run(func(line string) {
			select {
			case ch <- line:
			default:
			}
		})
		close(ch)
		return brewDoneMsg{action: action, err: err}
	}
	waiter := func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return nil
		}
		return brewLineMsg{line: line}
	}
	return runner, waiter
}

// runIntegrationCmd runs wizard.InstallMemcli or wizard.UninstallMemcli based
// on install, producing an integrationDoneMsg when done.
func runIntegrationCmd(prov provider.Provider, install bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		action := "install integration"
		if install {
			err = wizard.InstallMemcli(prov)
		} else {
			err = wizard.UninstallMemcli(prov)
			action = "uninstall integration"
		}
		return integrationDoneMsg{action: action, err: err}
	}
}

// popCmd is the canonical emitter for "return to caller".
func popCmd() tea.Cmd {
	return func() tea.Msg { return messages.PopScreenMsg{} }
}

// appendTail appends line to buf keeping at most max entries.
func appendTail(buf []string, line string, max int) []string {
	buf = append(buf, line)
	if len(buf) > max {
		buf = buf[len(buf)-max:]
	}
	return buf
}
