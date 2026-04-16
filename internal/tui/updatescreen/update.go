// Package updatescreen implements the Update screen for dotai.
// It runs a two-phase update: (1) upgrade the dotai binary via brew (skipped
// when brew is unavailable or dotai is not a brew formula) and (2) re-apply
// the embedded templates to the components the user already has installed.
// When brew actually replaces the binary, the process re-executes itself so
// that phase 2 runs under the freshly-installed code and asset set.
package updatescreen

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

// phase tracks where in the pipeline the screen currently is.
type phase int

const (
	phaseLoading  phase = iota // async detect in progress
	phasePlan                  // preview + action buttons
	phaseBrew                  // brew update/upgrade streaming
	phaseFiles                 // wizard.Update re-apply
	phaseDone                  // final report
)

// brewOutputTail is how many trailing brew stdout/stderr lines we keep on
// screen while the upgrade is running.
const brewOutputTail = 6

// planMsg delivers the detection snapshot, the preview rows and the brew
// availability back to the model after the async Init pass.
type planMsg struct {
	preview []previewRow
	brew    selfupdate.Availability
}

// previewRow is one line of the "will update" list shown before confirmation.
type previewRow struct {
	label     string
	installed bool
	detail    string
}

// brewLineMsg delivers a single stdout/stderr line from the running brew
// sub-process so the UI can append it to the live tail.
type brewLineMsg struct {
	line string
}

// brewDoneMsg signals that the brew update/upgrade sequence has finished.
type brewDoneMsg struct {
	result selfupdate.RunResult
}

// updateDoneMsg signals the file-reapply phase has finished.
type updateDoneMsg struct {
	report wizard.UpdateReport
	err    error
}

// relaunchFailedMsg is emitted when RelaunchSelfWith returns (which only
// happens on failure — a successful exec never returns).
type relaunchFailedMsg struct {
	err error
}

// Model is the Bubble Tea model for the update screen.
type Model struct {
	prov          provider.Provider
	phase         phase
	preview       []previewRow
	brew          selfupdate.Availability
	brewTail      []string
	brewWait      tea.Cmd
	report        wizard.UpdateReport
	toast         string
	toastErr      bool
	cursor        int
	resumed       bool
	width, height int
}

// New returns an update Model for a regular (menu-driven) entry.
func New(prov provider.Provider) Model {
	return Model{prov: prov, phase: phaseLoading}
}

// NewResumed returns an update Model that was entered automatically after the
// binary re-executed itself post-brew-upgrade. It skips the plan preview and
// jumps straight to the file-reapply phase; the user only sees "applying..."
// and then the final report.
func NewResumed(prov provider.Provider) Model {
	return Model{prov: prov, phase: phaseLoading, resumed: true}
}

// Title satisfies messages.Screen.
func (m Model) Title() string { return "update" }

// Init triggers an async detection pass that feeds the preview list and the
// brew availability.
func (m Model) Init() tea.Cmd {
	prov := m.prov
	return func() tea.Msg {
		return planMsg{
			preview: buildPreview(prov),
			brew:    selfupdate.Detect(),
		}
	}
}

// Update handles messages and key events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case planMsg:
		m.preview = msg.preview
		m.brew = msg.brew
		m.phase = phasePlan
		// On a resumed entry, the brew phase already ran in the previous
		// process — jump straight to file re-apply.
		if m.resumed {
			m.phase = phaseFiles
			return m, runFilesCmd(m.prov)
		}
		return m, nil

	case brewLineMsg:
		m.brewTail = appendTail(m.brewTail, msg.line, brewOutputTail)
		return m, m.brewWait

	case brewDoneMsg:
		if msg.result.ExitErr != nil {
			m.phase = phaseDone
			m.toast = "brew upgrade failed: " + msg.result.ExitErr.Error()
			m.toastErr = true
			return m, nil
		}
		if msg.result.UpdatedDotai {
			// The on-disk binary was replaced. Re-exec so the new code
			// (with its new embedded assets) performs the file re-apply.
			return m, relaunchCmd()
		}
		// Nothing to upgrade — proceed with file re-apply in this process.
		m.phase = phaseFiles
		return m, runFilesCmd(m.prov)

	case relaunchFailedMsg:
		// Exec failed — fall back to re-applying with the current (old)
		// binary and warn the user.
		m.phase = phaseFiles
		m.toast = "self-relaunch failed: " + msg.err.Error() + " — applying with current binary"
		m.toastErr = true
		return m, runFilesCmd(m.prov)

	case updateDoneMsg:
		m.phase = phaseDone
		m.report = msg.report
		if msg.err != nil {
			m.toast = "update failed: " + msg.err.Error()
			m.toastErr = true
		} else if !m.toastErr {
			// Preserve any earlier warning toast (e.g. relaunch-failed
			// fallback); only set the success toast when nothing important
			// is already on screen.
			m.toast = "update completed"
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// handleKey routes keyboard input based on the current phase. Input is
// ignored while busy phases (brew / files) are running.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.phase == phaseBrew || m.phase == phaseFiles || m.phase == phaseLoading {
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < 1 {
			m.cursor++
		}
	case "enter":
		return m.confirmAction()
	case "esc", "q":
		return m, popScreen()
	}
	return m, nil
}

// confirmAction runs the currently selected button. On phasePlan: cursor 0
// starts the update, cursor 1 pops back to the menu. On phaseDone: any cursor
// pops back.
func (m Model) confirmAction() (tea.Model, tea.Cmd) {
	if m.phase == phaseDone {
		return m, popScreen()
	}
	switch m.cursor {
	case 0:
		if !anyInstalled(m.preview) && !m.brew.CanUpgrade() {
			return m, popScreen()
		}
		if m.brew.CanUpgrade() {
			m.phase = phaseBrew
			m.brewTail = nil
			runner, waiter := brewCmds()
			m.brewWait = waiter
			return m, tea.Batch(runner, waiter)
		}
		m.phase = phaseFiles
		return m, runFilesCmd(m.prov)
	case 1:
		return m, popScreen()
	}
	return m, nil
}

// SetSize stores the terminal dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// View renders the update screen inside a FrameCompact border.
func (m Model) View() string {
	innerWidth := styles.CompactInnerWidth
	header := banner.RenderCompact(innerWidth, "update")

	var body strings.Builder
	switch m.phase {
	case phaseLoading:
		body.WriteString(styles.Help.Render("Detecting installed components..."))
	case phasePlan:
		body.WriteString(renderPlan(m.preview, m.brew))
		body.WriteString("\n")
		body.WriteString(renderActions(m.cursor, false))
	case phaseBrew:
		body.WriteString(styles.SectionTitle.Render("Upgrading dotai via brew..."))
		body.WriteString("\n\n")
		body.WriteString(renderBrewTail(m.brewTail))
	case phaseFiles:
		body.WriteString(styles.SectionTitle.Render("Applying embedded templates..."))
	case phaseDone:
		body.WriteString(renderReport(m.report))
		body.WriteString("\n")
		body.WriteString(renderActions(m.cursor, true))
	}

	if m.toast != "" && m.phase == phaseDone {
		body.WriteString("\n")
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
	if m.phase == phaseBrew || m.phase == phaseFiles || m.phase == phaseLoading {
		return styles.FooterHints("please wait...", "")
	}
	return styles.FooterHints("↑/↓", "navigate", "enter", "select", "esc", "back")
}

// renderPlan renders the pre-update preview: the brew row (if available)
// followed by the detected file components.
func renderPlan(rows []previewRow, brew selfupdate.Availability) string {
	var b strings.Builder
	b.WriteString(styles.SectionTitle.Render("What will be updated:"))
	b.WriteString("\n\n")

	if brew.CanUpgrade() {
		b.WriteString(styles.CheckOn.Render("✓") + " " +
			styles.MenuItem.Render("dotai binary") + "  " +
			styles.Dim("brew update && brew upgrade dotai"))
		b.WriteString("\n")
	} else {
		b.WriteString(styles.CheckOff.Render("✗") + " " +
			styles.Help.Render("dotai binary (not managed by brew — skipped)"))
		b.WriteString("\n")
	}

	anyInst := false
	for _, row := range rows {
		if row.installed {
			anyInst = true
			b.WriteString(styles.CheckOn.Render("✓") + " " +
				styles.MenuItem.Render(row.label))
			if row.detail != "" {
				b.WriteString("  " + styles.Dim(row.detail))
			}
			b.WriteString("\n")
		} else {
			b.WriteString(styles.CheckOff.Render("✗") + " " +
				styles.Help.Render(row.label+" (not installed — skipped)"))
			b.WriteString("\n")
		}
	}
	if !anyInst && !brew.CanUpgrade() {
		b.WriteString("\n")
		b.WriteString(styles.WarningText.Render("Nothing to update — run Install first."))
	}
	return b.String()
}

// renderBrewTail prints the last few lines of brew output so the user sees
// live progress while the command is blocking.
func renderBrewTail(lines []string) string {
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

// renderActions renders the two bottom buttons (Run Update / Back).
// When done is true, the primary button becomes "Done" alone — the update
// can only be run once per session.
func renderActions(cursor int, done bool) string {
	var b strings.Builder
	primary := "Run Update"
	if done {
		primary = "Done"
	}
	labels := []string{primary, "Back"}
	for i, label := range labels {
		if i == cursor {
			b.WriteString(styles.Arrow.Render("▸ ") + styles.MenuItemSelected.Render(label))
		} else {
			b.WriteString("  " + styles.MenuItem.Render(label))
		}
		if i < len(labels)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderReport renders the post-update summary: updated rows first, then any
// skipped rows with their reason.
func renderReport(r wizard.UpdateReport) string {
	var b strings.Builder
	b.WriteString(styles.SectionTitle.Render("Updated:"))
	b.WriteString("\n")
	if len(r.Updated) == 0 {
		b.WriteString(styles.Help.Render("  (nothing)"))
		b.WriteString("\n")
	}
	for _, item := range r.Updated {
		b.WriteString(styles.CheckOn.Render("  ✓ ") + styles.MenuItem.Render(item))
		b.WriteString("\n")
	}
	if len(r.Skipped) > 0 {
		b.WriteString("\n")
		b.WriteString(styles.SectionTitle.Render("Skipped:"))
		b.WriteString("\n")
		for _, s := range r.Skipped {
			b.WriteString(styles.CheckOff.Render("  ✗ ") +
				styles.Help.Render(s.Component+" — "+s.Reason))
			b.WriteString("\n")
		}
	}
	return b.String()
}

// buildPreview runs CheckAll and converts the result into preview rows in a
// stable display order. Shown to the user before confirmation.
func buildPreview(prov provider.Provider) []previewRow {
	results := detect.CheckAll(prov)
	byComponent := make(map[detect.Component]detect.CheckResult, len(results))
	for _, r := range results {
		byComponent[r.Component] = r
	}

	order := []struct {
		comp  detect.Component
		label string
	}{
		{detect.Profile, "CLAUDE.md"},
		{detect.Settings, "settings.json"},
		{detect.Statusline, "statusline"},
		{detect.GitHook, "git-branch-check hook"},
		{detect.Skills, "skills"},
		{detect.Memcli, "memcli"},
	}

	rows := make([]previewRow, 0, len(order))
	for _, o := range order {
		r := byComponent[o.comp]
		rows = append(rows, previewRow{
			label:     o.label,
			installed: r.Installed,
			detail:    r.Detail,
		})
	}
	return rows
}

// anyInstalled reports whether at least one row in the preview is installed.
func anyInstalled(rows []previewRow) bool {
	for _, r := range rows {
		if r.installed {
			return true
		}
	}
	return false
}

// brewCmds returns a matched (runner, waiter) pair for one brew execution.
// The runner streams brew output into an internal channel and, when brew
// finishes, closes the channel and emits brewDoneMsg. The waiter blocks on
// the channel and emits brewLineMsg per line; once the channel is closed it
// returns nil, stopping the Bubble Tea loop from re-issuing itself. Each
// brew run uses its own channel so stale output from a prior run cannot
// bleed into a subsequent one.
func brewCmds() (tea.Cmd, tea.Cmd) {
	ch := make(chan string, 128)
	runner := func() tea.Msg {
		res := selfupdate.RunBrewUpgrade(func(line string) {
			select {
			case ch <- line:
			default:
				// Drop line on overflow — the tail only needs the last few
				// lines anyway and we would rather lose noise than block.
			}
		})
		close(ch)
		return brewDoneMsg{result: res}
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

// runFilesCmd invokes wizard.Update asynchronously and reports the outcome.
func runFilesCmd(prov provider.Provider) tea.Cmd {
	return func() tea.Msg {
		rep, err := wizard.Update(prov)
		return updateDoneMsg{report: rep, err: err}
	}
}

// relaunchCmd attempts to replace the current process with a fresh dotai
// invocation, passing --resume-update so the new process jumps directly into
// the file-reapply phase. On success this never returns; on failure it
// reports the error so the screen can fall back gracefully.
func relaunchCmd() tea.Cmd {
	return func() tea.Msg {
		err := selfupdate.RelaunchSelfWith("--resume-update")
		return relaunchFailedMsg{err: err}
	}
}

// popScreen is the standard PopScreenMsg emitter shared by every exit path.
func popScreen() tea.Cmd {
	return func() tea.Msg { return messages.PopScreenMsg{} }
}

// appendTail appends line to buf and trims from the head to keep at most
// max entries. It is safe to call with buf==nil.
func appendTail(buf []string, line string, max int) []string {
	buf = append(buf, line)
	if len(buf) > max {
		buf = buf[len(buf)-max:]
	}
	return buf
}
