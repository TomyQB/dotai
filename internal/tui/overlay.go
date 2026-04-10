package tui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/mem-cli/internal/doctor"
	"github.com/TomyQB/mem-cli/internal/tui/messages"
	"github.com/TomyQB/mem-cli/internal/tui/preview"
	"github.com/TomyQB/mem-cli/internal/tui/project"
	"github.com/TomyQB/mem-cli/internal/tui/styles"
)

// pushProject pushes a Project screen for the selected entry.
func (m rootModel) pushProject(msg messages.ProjectSelectedMsg) rootModel {
	return m.pushScreen(project.New(msg.Entry))
}

// pushPreview pushes a Preview screen for the selected doc.
func (m rootModel) pushPreview(msg messages.DocSelectedMsg) (rootModel, tea.Cmd) {
	m = m.pushScreen(preview.New(msg.Path))
	return m, nil
}

// pushScreen pre-sizes a screen with the current terminal dimensions and
// appends it to the stack. Extracted to DRY the push pattern.
func (m rootModel) pushScreen(s messages.Screen) rootModel {
	if m.width > 0 && m.height > 0 {
		updated, _ := s.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		if sized, ok := updated.(messages.Screen); ok {
			m.stack = append(m.stack, sized)
			return m
		}
	}
	m.stack = append(m.stack, s)
	return m
}

// doctorModal is the overlay shown while a doctor report is visible.
type doctorModal struct {
	vp   viewport.Model
	body string // cached — report never changes after creation
}

func newDoctorModal(report doctor.Report, err error) doctorModal {
	body := report.String()
	if err != nil {
		body = "doctor error:\n\n" + err.Error()
	}
	vp := viewport.New(styles.CompactInnerWidth, 15)
	vp.SetContent(body)
	return doctorModal{vp: vp, body: body}
}

func (d doctorModal) Init() tea.Cmd { return nil }

func (d doctorModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.WindowSizeMsg:
		// Compact modal: fixed size, no resize handling needed.
		return d, nil
	}
	var cmd tea.Cmd
	d.vp, cmd = d.vp.Update(msg)
	return d, cmd
}

func (d doctorModal) View() string {
	header := styles.SectionTitle.Render("Doctor")
	footer := styles.FooterHints("esc", "close")
	body := d.vp.View()
	return styles.FrameCompact(header, body+"\n\npress esc to close", footer, styles.CompactWidth)
}
