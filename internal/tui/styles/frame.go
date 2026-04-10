package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Frame composes header/body/footer inside the global rounded violet Border
// and sizes the result to width x height. Any of the three sections may be
// empty. Body is top-aligned; footer is pinned to the bottom of the body area.
//
// Frame is used by fullscreen screens (Preview) where we want the content to
// stretch to fill the terminal.
func Frame(header, body, footer string, width, height int) string {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}

	// Reserve space for the border (2) and the horizontal padding (4).
	innerW := width - 6
	if innerW < 10 {
		innerW = 10
	}
	// Border + vertical padding (2).
	innerH := height - 4
	if innerH < 4 {
		innerH = 4
	}

	sections := []string{}
	if header != "" {
		sections = append(sections, header, "")
	}

	headerH := lipgloss.Height(strings.Join(sections, "\n"))
	footerH := 0
	if footer != "" {
		footerH = lipgloss.Height(footer) + 1 // +1 for blank line above
	}

	bodyH := innerH - headerH - footerH
	if bodyH < 1 {
		bodyH = 1
	}

	bodyBlock := lipgloss.NewStyle().
		Width(innerW).
		Height(bodyH).
		Render(body)
	sections = append(sections, bodyBlock)

	if footer != "" {
		sections = append(sections, "", lipgloss.PlaceHorizontal(innerW, lipgloss.Left, footer))
	}

	content := strings.Join(sections, "\n")
	return Border.Width(innerW).Render(content)
}

// FrameCompact renders header/body/footer inside the violet border at a fixed
// maxWidth. Height wraps tightly around content — no stretching.
// This is the frame used by Menu, Home, Project, and Doctor.
func FrameCompact(header, body, footer string, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 50
	}
	innerW := maxWidth - 6 // border (2) + padding (4)
	if innerW < 10 {
		innerW = 10
	}

	sections := []string{}
	if header != "" {
		sections = append(sections, header, "")
	}
	sections = append(sections, body)
	if footer != "" {
		sections = append(sections, "", lipgloss.PlaceHorizontal(innerW, lipgloss.Left, footer))
	}
	content := strings.Join(sections, "\n")
	return Border.Width(innerW).Render(content)
}

// FooterHints renders a "key: desc • key: desc ..." footer line using the
// HelpKey / HelpDesc / FooterSep styles. hints must be pairs: [k1,d1,k2,d2,...].
func FooterHints(hints ...string) string {
	var b strings.Builder
	for i := 0; i+1 < len(hints); i += 2 {
		if i > 0 {
			b.WriteString(FooterSep.Render(" • "))
		}
		b.WriteString(HelpKey.Render(hints[i]))
		b.WriteString(HelpDesc.Render(": " + hints[i+1]))
	}
	return b.String()
}
