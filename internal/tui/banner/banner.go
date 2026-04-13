// Package banner renders the dotai ASCII banner. It has no dependencies on
// any figlet library — the big block letters and the brain art are embedded
// as raw string constants. Render() gracefully degrades as width shrinks.
package banner

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/TomyQB/dotai/internal/tui/styles"
	"github.com/TomyQB/dotai/internal/version"
)

const tagline = "AI coding environment, one shot"

// brain is the small ASCII brain shown above the wordmark. 7 lines, 28 cols.
const brain = `      ___-----___
    .-'  .---.  '-.
   /    /     \    \
  |    | () () |    |
   \    \  ^  /    /
    '-.__'---'__.-'
        /___\`

// wordmark is the "dotai" ANSI Shadow figlet. 6 lines, 39 cols wide.
const wordmark = `██████╗  ██████╗ ████████╗ █████╗ ██╗
██╔══██╗██╔═══██╗╚══██╔══╝██╔══██╗██║
██║  ██║██║   ██║   ██║   ███████║██║
██║  ██║██║   ██║   ██║   ██╔══██║██║
██████╔╝╚██████╔╝   ██║   ██║  ██║██║
╚═════╝  ╚═════╝    ╚═╝   ╚═╝  ╚═╝╚═╝`

// wordmarkSmall is a compact box-drawing version that fits in ~15 chars.
const wordmarkSmall = `╺┳┓┏━┓╺┳╸┏━┓╻
 ┃┃┃ ┃ ┃ ┣━┫┃
╺┻┛┗━┛ ╹ ╹ ╹╹`

// brainSmall is a compact 4-line brain for compact screens.
const brainSmall = `    .---.
   / o o \
   \ -^- /
    '---'`

const (
	wordmarkWidth      = 39
	brainWidth         = 28
	wordmarkSmallWidth = 15
	brainSmallWidth    = 12
)

// Render returns the banner for compact (inline) screens. Uses the small
// wordmark and small brain. The output is centered inside width.
func Render(width int) string {
	if width <= 0 {
		return ""
	}
	if width < wordmarkSmallWidth {
		return renderCompact(width)
	}
	parts := []string{}
	if width >= wordmarkSmallWidth+4 {
		parts = append(parts, center(styles.BannerAccent.Render(brainSmall), width))
	}
	parts = append(parts,
		center(styles.Banner.Render(wordmarkSmall), width),
		center(styles.Tagline.Render("v"+version.Version+" — "+tagline), width),
	)
	return strings.Join(parts, "\n")
}

// RenderFull returns the full large banner for fullscreen contexts (e.g. Preview).
// Degradation order:
//
//  1. width >= wordmarkWidth+4: brain + wordmark + tagline (full banner)
//  2. width >= wordmarkWidth:   wordmark + tagline (no brain)
//  3. otherwise:                 compact title "dotai v0.1.0"
func RenderFull(width int) string {
	if width <= 0 {
		return ""
	}
	if width < wordmarkWidth {
		return renderCompact(width)
	}
	parts := []string{}
	if width >= wordmarkWidth+4 {
		parts = append(parts, center(styles.BannerAccent.Render(brain), width))
	}
	parts = append(parts,
		center(styles.Banner.Render(wordmark), width),
		center(styles.Tagline.Render("v"+version.Version+" — "+tagline), width),
	)
	return strings.Join(parts, "\n")
}

// RenderCompact returns a single-line compact header suitable for sub-screens:
// "dotai · <screen>". Width is used only for centering.
func RenderCompact(width int, screen string) string {
	title := "dotai"
	if screen != "" {
		title += " · " + screen
	}
	return center(styles.Title.Render(title), width)
}

func renderCompact(width int) string {
	return center(styles.Title.Render("dotai v"+version.Version), width)
}

// center horizontally centers a (possibly multiline) block inside width,
// without clobbering ANSI sequences. Uses lipgloss.PlaceHorizontal.
func center(block string, width int) string {
	if width <= 0 {
		return block
	}
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, block)
}
