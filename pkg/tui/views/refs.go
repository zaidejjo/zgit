package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/components"
	"github.com/zaidejjo/zgit/pkg/tui/glyphs"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// RefsSection selects the focused half of the Refs screen.
type RefsSection int

const (
	RefsTags RefsSection = iota
	RefsRemotes
)

// RefsModel holds tags + remotes with a focus toggle (t key).
type RefsModel struct {
	Tags    components.ZList[string]
	Remotes components.ZList[*models.Remote]
	Section RefsSection
	Loaded  bool
	Error   string
	Prompt  Prompt
}

// NewRefsModel creates the refs view model.
func NewRefsModel() RefsModel {
	m := RefsModel{}
	m.Tags.Height = 20
	m.Tags.Match = func(tag, q string) bool {
		return strings.Contains(strings.ToLower(tag), q)
	}
	m.Tags.Render = tagRow
	m.Remotes.Height = 20
	m.Remotes.Match = func(r *models.Remote, q string) bool {
		return strings.Contains(strings.ToLower(r.Name+" "+r.URL), q)
	}
	m.Remotes.Render = remoteRow
	return m
}

// UpdateRefs refreshes both lists.
func (m *RefsModel) UpdateRefs(tags []string, remotes []*models.Remote) {
	m.Tags.Items = tags
	m.Remotes.Items = remotes
	m.Loaded = true
	m.Error = ""
	m.Tags.ClampCursor()
	m.Remotes.ClampCursor()
}

// Toggle flips the focused section.
func (m *RefsModel) Toggle() {
	if m.Section == RefsTags {
		m.Section = RefsRemotes
	} else {
		m.Section = RefsTags
	}
}

// View renders tags then remotes under one screen chrome; each half keeps
// its own section label so the focus toggle stays readable.
func (m *RefsModel) View(width int) string {
	body := m.body(width)
	meta := fmt.Sprintf("%d tags · %d remotes", len(m.Tags.Items), len(m.Remotes.Items))
	return screenChrome("Refs", meta, body, width)
}

func (m *RefsModel) body(width int) string {
	if m.Error != "" {
		return styles.ErrorStyle.Render("Error: " + m.Error)
	}
	if !m.Loaded {
		return styles.LoadingStyle.Render("Loading tags & remotes...")
	}

	var b strings.Builder
	b.WriteString(m.sectionTitle("Tags", len(m.Tags.Items), m.Section == RefsTags))
	b.WriteString("\n")
	if m.Section == RefsTags {
		b.WriteString(m.Tags.FilterBar())
	}
	if len(m.Tags.Items) == 0 {
		b.WriteString(styles.SubtitleStyle.Render(" No tags — press c to create one"))
		b.WriteString("\n")
	} else {
		b.WriteString(m.Tags.ViewRows(width))
	}
	b.WriteString("\n")
	b.WriteString(m.sectionTitle("Remotes", len(m.Remotes.Items), m.Section == RefsRemotes))
	b.WriteString("\n")
	if m.Section == RefsRemotes {
		b.WriteString(m.Remotes.FilterBar())
	}
	if len(m.Remotes.Items) == 0 {
		b.WriteString(styles.SubtitleStyle.Render(" No remotes configured"))
		b.WriteString("\n")
	} else {
		b.WriteString(m.Remotes.ViewRows(width))
	}
	return b.String()
}

// sectionTitle is a section label with its count, the way the status pane
// labels STAGED and UNSTAGED: uppercase, quiet, count attached.
func (m *RefsModel) sectionTitle(name string, n int, focused bool) string {
	st := styles.SectionLabelStyle
	if focused {
		st = styles.PaneTitleFocusStyle
	}
	return " " + st.Render(strings.ToUpper(name)) + " " + styles.MetaStyle.Render(fmt.Sprintf("%d", n))
}

func tagRow(tag string, width int) string {
	return components.Truncate(
		fmt.Sprintf("  %s %s", styles.SubtitleStyle.Render(glyphs.Current().Tag), truncateStr(tag, width-6)),
		width)
}

func remoteRow(r *models.Remote, width int) string {
	line := fmt.Sprintf("  %s %s  %s",
		styles.StatusStagedStyle.Render(glyphs.Current().Dot),
		truncateStr(r.Name, 24), styles.SubtitleStyle.Render(truncateStr(r.URL, 50)))
	if lipgloss.Width(line) > width {
		line = fmt.Sprintf("  %s %s",
			styles.StatusStagedStyle.Render(glyphs.Current().Dot), truncateStr(r.Name, width-6))
	}
	return components.Truncate(line, width)
}
