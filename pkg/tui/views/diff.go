package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// DiffModel renders a syntax-highlighted unified diff with viewport scrolling.
type DiffModel struct {
	File      string
	DiffText  string
	Additions int
	Deletions int
	viewport  viewport.Model
	ready     bool
	width     int
	height    int
}

// NewDiffModel creates a default diff viewer model.
func NewDiffModel() DiffModel {
	return DiffModel{}
}

// SetDiff updates the diff content and resets viewport.
func (m *DiffModel) SetDiff(file string, diffText string, adds, dels int) {
	m.File = file
	m.DiffText = diffText
	m.Additions = adds
	m.Deletions = dels
	m.ready = false
}

// Clear resets the diff viewer.
func (m *DiffModel) Clear() {
	m.File = ""
	m.DiffText = ""
	m.Additions = 0
	m.Deletions = 0
	m.ready = false
}

// Active returns true when a diff is loaded.
func (m *DiffModel) Active() bool {
	return m.File != ""
}

// Init implements tea.Model.
func (m DiffModel) Init() tea.Cmd { return nil }

// Update handles key events and viewport resizing.
func (m *DiffModel) Update(msg tea.Msg) (*DiffModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = false
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.viewport.LineDown(1)
		case "k", "up":
			m.viewport.LineUp(1)
		case "pgdown", "ctrl+f":
			m.viewport.HalfViewDown()
		case "pgup", "ctrl+b":
			m.viewport.HalfViewUp()
		case "g", "home":
			m.viewport.GotoTop()
		case "G", "end":
			m.viewport.GotoBottom()
		}
	}
	return m, nil
}

// View renders the diff content inside a viewport.
func (m *DiffModel) View(width int) string {
	if !m.Active() {
		return ""
	}

	if !m.ready {
		m.viewport = viewport.New(width-2, m.height-6)
		m.viewport.Style = lipgloss.NewStyle().PaddingLeft(1)
		m.viewport.SetContent(colorizeDiff(m.DiffText, width-4))
		m.ready = true
	}

	// Header
	header := styles.DialogTitleStyle.Render(fmt.Sprintf("  diff --git a/%s b/%s", m.File, m.File))
	stats := fmt.Sprintf("  +%d  -%d", m.Additions, m.Deletions)
	statsRendered := lipgloss.NewStyle().
		Foreground(styles.Green).
		Render(strings.Repeat("+", m.Additions)) +
		lipgloss.NewStyle().
			Foreground(styles.Red).
			Render(strings.Repeat("-", m.Deletions))

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(statsRendered)
	b.WriteString(fmt.Sprintf("  %s\n\n", stats))
	b.WriteString(m.viewport.View())

	// Footer with scroll info
	percent := m.viewport.ScrollPercent()
	scroll := fmt.Sprintf("  ↑/↓ scroll  Esc close  %.0f%%", percent*100)
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(styles.Subtext).Render(scroll))

	return b.String()
}

// colorizeDiff applies syntax highlighting to unified diff text.
func colorizeDiff(diff string, maxWidth int) string {
	lines := strings.Split(diff, "\n")
	var colored []string

	for _, line := range lines {
		if maxWidth > 0 && len(line) > maxWidth {
			line = line[:maxWidth]
		}

		switch {
		case strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- "):
			colored = append(colored, styles.DiffHeaderStyle.Render(line))
		case strings.HasPrefix(line, "@@"):
			colored = append(colored, styles.DiffHeaderStyle.Render(line))
		case strings.HasPrefix(line, "+"):
			colored = append(colored, styles.DiffAddStyle.Render(line))
		case strings.HasPrefix(line, "-"):
			colored = append(colored, styles.DiffDelStyle.Render(line))
		case strings.HasPrefix(line, "diff --git"):
			colored = append(colored, styles.SectionTitleStyle.Render(line))
		default:
			colored = append(colored, line)
		}
	}
	return strings.Join(colored, "\n")
}
