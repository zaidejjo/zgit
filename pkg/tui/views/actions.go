package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// ActionsModel holds state for the CI workflow runs view. Enter drills
// into the selected run's jobs; backspace returns to the run list.
type ActionsModel struct {
	Runs      []*models.WorkflowRun
	Jobs      []*models.Job
	JobsFor   int64 // run id whose jobs are shown, 0 = run list
	Cursor    int
	Offset    int
	Error     string
	Height    int
	Filter    string
	Filtering bool
}

// NewActionsModel creates the actions view model.
func NewActionsModel() ActionsModel {
	return ActionsModel{Height: 20}
}

// UpdateRuns refreshes the run list (clears any job drill-in).
func (m *ActionsModel) UpdateRuns(runs []*models.WorkflowRun) {
	m.Runs = runs
	m.Jobs = nil
	m.JobsFor = 0
	m.Error = ""
	m.ClampCursor()
}

// UpdateJobs shows jobs for a run.
func (m *ActionsModel) UpdateJobs(runID int64, jobs []*models.Job) {
	m.Jobs = jobs
	m.JobsFor = runID
	m.Cursor = 0
	m.Offset = 0
	m.Error = ""
}

// Filtered returns runs matching the filter (workflow/branch/sha/event).
func (m *ActionsModel) Filtered() []*models.WorkflowRun {
	q := strings.ToLower(strings.TrimSpace(m.Filter))
	if q == "" {
		return m.Runs
	}
	out := make([]*models.WorkflowRun, 0, len(m.Runs))
	for _, r := range m.Runs {
		hay := strings.ToLower(strings.Join([]string{
			r.WorkflowName, r.Branch, r.HeadSHA, r.Event,
			fmt.Sprintf("%d", r.RunNumber),
		}, " "))
		if strings.Contains(hay, q) {
			out = append(out, r)
		}
	}
	return out
}

// ClampCursor keeps the cursor inside the visible list.
func (m *ActionsModel) ClampCursor() {
	n := len(m.Filtered())
	if m.JobsFor != 0 {
		n = len(m.Jobs)
	}
	if m.Cursor >= n {
		m.Cursor = 0
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}
}

// SelectedRun returns the cursor run (filter-aware), or nil in job view.
func (m *ActionsModel) SelectedRun() *models.WorkflowRun {
	if m.JobsFor != 0 {
		return nil
	}
	visible := m.Filtered()
	if len(visible) == 0 || m.Cursor < 0 || m.Cursor >= len(visible) {
		return nil
	}
	return visible[m.Cursor]
}

func runMark(r *models.WorkflowRun) string {
	c := strings.ToLower(r.Conclusion)
	switch c {
	case "success":
		return styles.StatusStagedStyle.Render("●")
	case "failure":
		return styles.StatusDeletedStyle.Render("●")
	case "cancelled", "skipped":
		return styles.StatusUntrackedStyle.Render("●")
	}
	if strings.ToLower(string(r.Status)) == "in_progress" {
		return styles.StatusUnstagedStyle.Render("◉")
	}
	return styles.SubtitleStyle.Render("○")
}

func jobMark(j *models.Job) string {
	c := strings.ToLower(j.Conclusion)
	switch c {
	case "success":
		return styles.StatusStagedStyle.Render("✓")
	case "failure":
		return styles.StatusDeletedStyle.Render("✗")
	case "cancelled", "skipped":
		return styles.StatusUntrackedStyle.Render("−")
	}
	if j.Status == "in_progress" {
		return styles.StatusUnstagedStyle.Render("◉")
	}
	return styles.SubtitleStyle.Render("○")
}

// View renders runs or, when drilled in, one run's jobs.
func (m *ActionsModel) View(width int) string {
	if m.Error != "" {
		return styles.ErrorStyle.Render("Error: " + m.Error)
	}
	if m.Runs == nil {
		return styles.LoadingStyle.Render("Loading workflow runs...")
	}
	if m.JobsFor != 0 {
		return m.jobsView(width)
	}

	if len(m.Runs) == 0 {
		return styles.SubtitleStyle.Render("No workflow runs found")
	}

	m.ensureCursorVisible()

	var b strings.Builder
	title := fmt.Sprintf(" Actions (%d)", len(m.Runs))
	if strings.TrimSpace(m.Filter) != "" {
		title = fmt.Sprintf(" Actions (%d/%d) filter: %s", len(m.Filtered()), len(m.Runs), m.Filter)
	}
	b.WriteString(styles.SectionTitleStyle.Render(title))
	b.WriteString("\n")
	if m.Filtering {
		b.WriteString(styles.FilterBarStyle.Render(" /" + m.Filter + "█"))
		b.WriteString("\n")
	}

	visible := m.Filtered()
	if len(visible) == 0 && strings.TrimSpace(m.Filter) != "" {
		return b.String() + styles.SubtitleStyle.Render("No runs match filter")
	}
	if len(visible) > m.Height {
		end := m.Offset + m.Height
		if end > len(visible) {
			end = len(visible)
		}
		visible = visible[m.Offset:end]
	}

	for i, r := range visible {
		globalIdx := m.Offset + i
		head := r.HeadSHA
		if len(head) > 7 {
			head = head[:7]
		}
		line := fmt.Sprintf(" %s %s  #%d %s %s",
			runMark(r), truncateStr(r.WorkflowName, 28),
			r.RunNumber, truncateStr(r.Branch, 24), head)
		if lipgloss.Width(line) > width {
			line = fmt.Sprintf(" %s %s #%d", runMark(r), truncateStr(r.WorkflowName, width-12), r.RunNumber)
		}
		if globalIdx == m.Cursor {
			b.WriteString(styles.ListItemActiveStyle.Render("❯" + line[1:]))
		} else {
			b.WriteString(styles.ListItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *ActionsModel) jobsView(width int) string {
	var b strings.Builder
	b.WriteString(styles.SectionTitleStyle.Render(fmt.Sprintf(" Jobs (%d) — backspace to return", len(m.Jobs))))
	b.WriteString("\n")
	if len(m.Jobs) == 0 {
		return b.String() + styles.SubtitleStyle.Render("No jobs reported for this run")
	}
	jobs := m.Jobs
	if len(jobs) > m.Height {
		end := m.Offset + m.Height
		if end > len(jobs) {
			end = len(jobs)
		}
		jobs = jobs[m.Offset:end]
	}
	for i, j := range jobs {
		globalIdx := m.Offset + i
		line := fmt.Sprintf(" %s %s", jobMark(j), truncateStr(j.Name, width-6))
		if globalIdx == m.Cursor {
			b.WriteString(styles.ListItemActiveStyle.Render("❯" + line[1:]))
		} else {
			b.WriteString(styles.ListItemStyle.Render(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m *ActionsModel) ensureCursorVisible() {
	m.ClampCursor()
	if m.Cursor < m.Offset {
		m.Offset = m.Cursor
	}
	if m.Cursor >= m.Offset+m.Height {
		m.Offset = m.Cursor - m.Height + 1
	}
}
