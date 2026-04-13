// Package styles centralizes all lipgloss styles used by TUI screens.
// One source of truth makes theme tweaks mechanical and keeps screen
// packages focused on behavior.
//
// Palette: deep violet brand (mind/memory symbolism).
package styles

import "github.com/charmbracelet/lipgloss"

// CompactWidth is the fixed outer width used by all inline (non-fullscreen)
// screens: Menu, Home, Project, Doctor modal. Fits the small wordmark + brain
// + menu items with comfortable padding.
const CompactWidth = 55

// Palette — raw hex values exposed so other packages can reuse them.
const (
	ColorPrimary      = "#A78BFA" // violet 400 — accents, selected items, arrows
	ColorPrimaryBold  = "#7C3AED" // violet 600 — titles, border
	ColorHighlight    = "#C4B5FD" // violet 300 — soft highlight bg
	ColorMuted        = "#6B7280" // gray 500  — footer, hints
	ColorText         = "#E5E7EB" // gray 200  — default body
	ColorSuccess      = "#34D399" // emerald   — OK badges
	ColorWarning      = "#FBBF24" // amber     — STALE badges
	ColorDanger       = "#F87171" // red       — MISSING badges
	ColorSubtle       = "#9CA3AF" // gray 400  — secondary text

	// CompactInnerWidth is CompactWidth minus border (2) and padding (4).
	CompactInnerWidth = CompactWidth - 6

	// Badge status constants — use these instead of raw strings.
	StatusOK      = "OK"
	StatusStale   = "STALE"
	StatusMissing = "MISSING"
)

var (
	// Border is the global frame: violet rounded border with padding.
	Border = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorPrimaryBold)).
		Padding(1, 2)

	// Title is the bold primary title (e.g. "mem-cli · home").
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorPrimaryBold))

	// Banner is the style applied to the ASCII banner block letters.
	Banner = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorPrimary))

	// BannerAccent is used for the small brain / tagline above/below the banner.
	BannerAccent = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimaryBold))

	// Tagline is the dim version tagline under the banner.
	Tagline = lipgloss.NewStyle().
		Italic(true).
		Foreground(lipgloss.Color(ColorSubtle))

	// MenuItem is a non-selected menu row.
	MenuItem = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorText))

	// MenuItemSelected is the highlighted menu row.
	MenuItemSelected = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(ColorPrimary))

	// Arrow is the leading glyph on selected items.
	Arrow = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorPrimary))

	// SectionTitle is a bold label above a block (e.g. "Menu").
	SectionTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorPrimaryBold))

	// Footer is the muted bottom hint line.
	Footer = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorMuted))

	// FooterSep is the subtle bullet separator between footer keybindings.
	FooterSep = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSubtle))

	// HelpKey is the bright part of a help hint ("enter").
	HelpKey = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorPrimary))

	// HelpDesc is the description after a help hint (": select").
	HelpDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted))

	// StatusBar is the bottom status line background (kept for compat).
	StatusBar = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color(ColorText)).
			Background(lipgloss.Color(ColorPrimaryBold))

	// Help is dim help text.
	Help = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMuted))

	// ErrorBox wraps an inline error rendering.
	ErrorBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorDanger)).
			Padding(0, 1)

	// ModalBorder is the border used by the doctor modal.
	ModalBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorPrimaryBold)).
			Padding(1, 2)

	// Toast is a transient inline status message.
	Toast = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorSuccess))

	badgeOK = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0B0F")).
		Background(lipgloss.Color(ColorSuccess)).
		Padding(0, 1)
	badgeStale = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0B0B0F")).
			Background(lipgloss.Color(ColorWarning)).
			Padding(0, 1)
	badgeMissing = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0B0B0F")).
			Background(lipgloss.Color(ColorDanger)).
			Padding(0, 1)
	dim = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSubtle))

	// Wizard-specific styles.

	// StepProgress renders the "Step N/6" counter in the step header.
	StepProgress = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorPrimaryBold))

	// CheckOn renders a selected/enabled indicator in success green.
	CheckOn = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorSuccess))

	// CheckOff renders a deselected/disabled indicator in muted gray.
	CheckOff = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMuted))

	// SummaryCheck renders a confirmed item on the summary screen.
	SummaryCheck = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorSuccess))

	// SummarySkip renders a skipped item on the summary screen.
	SummarySkip = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMuted))

	// WarningText renders a bold warning message in amber.
	WarningText = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorWarning))

	// ProgressDone renders a completed progress indicator in success green.
	ProgressDone = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSuccess))

	// ProgressError renders a failed progress indicator in red, bold.
	ProgressError = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorDanger))

	// Cursor renders the selection cursor glyph in primary violet.
	Cursor = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorPrimary))
)

// BadgeOK renders the OK badge text.
func BadgeOK() string { return badgeOK.Render("OK") }

// BadgeStale renders the STALE badge text.
func BadgeStale() string { return badgeStale.Render("STALE") }

// BadgeMissing renders the MISSING badge text.
func BadgeMissing() string { return badgeMissing.Render("MISSING") }

// Badge returns the badge for the given status string: "OK", "STALE", "MISSING".
// Unknown values fall back to OK to avoid crashing on unexpected input.
func Badge(status string) string {
	switch status {
	case StatusStale:
		return BadgeStale()
	case StatusMissing:
		return BadgeMissing()
	default:
		return BadgeOK()
	}
}

// Dim renders dimmed text (used for paths in list rows).
func Dim(s string) string { return dim.Render(s) }
