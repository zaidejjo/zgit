package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/components"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// ActionsModel holds state for the CI workflow runs view. Enter drills
// into the selected run's jobs; backspace returns to the run list.
type ActionsModel struct {
	components.ZList[*models.WorkflowRun]
	Jobs    []*models.Job
	JobsFor int64 // run id whose jobs are shown, 0 = run list
	Error   string
}

// Runs aliases the underlying items.
func (m *ActionsModel) Runs() []*models.WorkflowRun {
	return m.Items
}

// NewActionsModel creates the actions view model.
func NewActionsModel() ActionsModel {
	m := ActionsModel{}
	m.Height = 20
	m.Match = func(r *models.WorkflowRun, q string) bool {
		hay := strings.ToLower(strings.Join([]string{
			r.WorkflowName, r.Branch, r.HeadSHA, r.Event,
			fmt.Sprintf("%d", r.RunNumber),
		}, " "))
		return strings.Contains(hay, q)
	}
	m.Render = actionsRow
	return m
}

// UpdateRuns refreshes the run list (clears any job drill-in).
func (m *ActionsModel) UpdateRuns(runs []*models.WorkflowRun) {
	m.Items = runs
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

// SelectedRun returns the cursor run (filter-aware), or nil in job view.
func (m *ActionsModel) SelectedRun() *models.WorkflowRun {
	if m.JobsFor != 0 {
		return nil
	}
	run, _ := m.Selected()
	return run
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
	if m.Items == nil {
		return styles.LoadingStyle.Render("Loading workflow runs...")
	}
	if m.JobsFor != 0 {
		return m.jobsView(width)
	}

	if len(m.Items) == 0 {
		return styles.SubtitleStyle.Render("No workflow runs found")
	}

	var b strings.Builder
	title := fmt.Sprintf(" Actions (%d)", len(m.Items))
	if strings.TrimSpace(m.Filter) != "" {
		title = fmt.Sprintf(" Actions (%d/%d) filter: %s", len(m.Filtered()), len(m.Items), m.Filter)
	}
	b.WriteString(styles.SectionTitleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(m.FilterBar())
	if len(m.Filtered()) == 0 && strings.TrimSpace(m.Filter) != "" {
		return b.String() + styles.SubtitleStyle.Render("No runs match filter")
	}

	return b.String() + m.ViewRows(width)
}

// actionsRow renders one run row (no cursor — ZList wraps it).
func actionsRow(r *models.WorkflowRun, width int) string {
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
	return line
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
		b.WriteString(components.RowLine(line, width, globalIdx == m.Cursor))
		b.WriteString("\n")
	}
	return b.String()
}
