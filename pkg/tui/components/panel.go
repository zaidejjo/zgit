package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// Panel renders a bordered container with a title bar.
// Focused state changes border color to accent.
// If Height > 0, total rendered height (including border) is fixed to Height rows.
type Panel struct {
	Title      string
	Subtitle   string // optional, shown after title in dimmed style
	Width      int
	Height     int // 0 = auto-height, > 0 = fixed height including border
	Focused    bool
	Content    string
	BorderType lipgloss.Border
}

// NewPanel creates a panel with default rounded border.
func NewPanel(title string, width int, focused bool) Panel {
	return Panel{
		Title:      title,
		Width:      width,
		Focused:    focused,
		BorderType: lipgloss.RoundedBorder(),
	}
}

// View renders the panel with border + title + content.
func (p Panel) View() string {
	if p.Width <= 4 {
		return ""
	}

	borderColor := styles.InactivePanelBorderColor
	if p.Focused {
		borderColor = styles.ActivePanelBorderColor
	}

	innerW := p.Width - 2 // minus left/right border

	// Build title line
	titleStr := p.Title
	if p.Subtitle != "" {
		titleStr = p.Title + " " + styles.SubtitleStyle.Render(p.Subtitle)
	}
	titleLine := lipgloss.NewStyle().
		Foreground(borderColor).
		Bold(p.Focused).
		Render(titleStr)

	// Body content — trim each line to inner width
	lines := strings.Split(p.Content, "\n")

	var bodyLines []string
	for _, line := range lines {
		trimmed := line
		// Use visual width (lipgloss.Width) not byte length (len).
		// len() overcounts because:
		//   - Multi-byte Unicode (● = 3 bytes, 1 visual col)
		//   - lipgloss ANSI codes (only present in TUI context)
		// → byte check falsely triggers truncation on fitting lines.
		// → byte-slicing breaks ANSI sequences and multi-byte chars.
		//
		// Visual width always ≤ innerW because views calculate content
		// to fit.  This check is a safety net — should never trigger.
		if lipgloss.Width(trimmed) > innerW {
			// Instead of byte-level truncation (corrupts ANSI/Unicode)
			// or Style.Width (wraps lines with newlines), we leave
			// overflow lines intact.  This is vanishingly rare since
			// views already bound content width.
		}
		bodyLines = append(bodyLines, trimmed)
	}

	// If Height is set, pad or truncate content to exact height
	if p.Height > 0 {
		// Inner height = total height - 2 (border top/bottom)
		innerH := p.Height - 2
		if innerH < 1 {
			innerH = 1
		}
		// First line is title, remaining lines are body
		maxBodyLines := innerH - 1
		if maxBodyLines < 0 {
			maxBodyLines = 0
		}
		if len(bodyLines) > maxBodyLines {
			bodyLines = bodyLines[:maxBodyLines]
		} else {
			// Pad with empty lines
			emptyLine := strings.Repeat(" ", innerW)
			for len(bodyLines) < maxBodyLines {
				bodyLines = append(bodyLines, emptyLine)
			}
		}
	}

	content := titleLine + "\n" + strings.Join(bodyLines, "\n")

	// Apply border
	return lipgloss.NewStyle().
		Border(p.BorderType).
		BorderForeground(borderColor).
		Width(p.Width).
		Render(content)
}
