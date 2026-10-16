package views

import (
	"testing"

	"github.com/zaidejjo/zgit/pkg/core/models"
)

func TestDashFiltered(t *testing.T) {
	m := NewDashModel()
	m.UpdateEntries([]DashEntry{
		{Name: "zgit", Path: "/r/zgit", Branch: "dev", Clean: true},
		{Name: "zz_lang", Path: "/r/zz_lang", Branch: "main", Clean: false, Dirty: 3},
	})
	if len(m.Filtered()) != 2 {
		t.Fatalf("unfiltered len = %d", len(m.Filtered()))
	}
	m.Filter = "zz_"
	if got := m.Filtered(); len(got) != 1 || got[0].Name != "zz_lang" {
		t.Fatalf("filtered = %+v", got)
	}
	m.Filter = "/R/ZGIT"
	if len(m.Filtered()) != 1 {
		t.Fatalf("case-insensitive path match failed")
	}
	m.Cursor = 9
	m.ClampCursor()
	if m.Cursor != 0 {
		t.Fatalf("clamp = %d", m.Cursor)
	}
	if sel := m.Selected(); sel == nil || sel.Name != "zgit" {
		t.Fatalf("selected = %+v", sel)
	}
}

func TestActionsFilteredAndJobs(t *testing.T) {
	m := NewActionsModel()
	m.UpdateRuns([]*models.WorkflowRun{
		{ID: 1, WorkflowName: "CI", Branch: "dev", HeadSHA: "abc1234", RunNumber: 10, Status: "COMPLETED", Conclusion: "SUCCESS"},
		{ID: 2, WorkflowName: "Docs", Branch: "main", HeadSHA: "def5678", RunNumber: 3, Status: "IN_PROGRESS"},
	})
	m.Filter = "docs"
	if got := m.Filtered(); len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("filtered = %+v", got)
	}
	m.Filter = ""
	if sel := m.SelectedRun(); sel == nil || sel.ID != 1 {
		t.Fatalf("selected = %+v", sel)
	}
	m.UpdateJobs(1, []*models.Job{{ID: 7, Name: "test", Status: "completed", Conclusion: "success"}})
	if m.JobsFor != 1 || len(m.Jobs) != 1 {
		t.Fatalf("drill-in state wrong: %+v", m)
	}
	if sel := m.SelectedRun(); sel != nil {
		t.Fatalf("SelectedRun in job view should be nil")
	}
}

func TestNotesUnreadFirst(t *testing.T) {
	m := NewNotesModel()
	m.UpdateNotes([]*models.Notification{
		{ID: "1", Title: "old", Unread: false, RepoName: "o/r"},
		{ID: "2", Title: "new", Unread: true, RepoName: "o/r"},
	})
	if got := m.UnreadCount(); got != 1 {
		t.Fatalf("unread = %d", got)
	}
	ordered := m.Ordered()
	if len(ordered) != 2 || ordered[0].ID != "2" {
		t.Fatalf("unread should sort first: %+v", ordered)
	}
	if sel := m.Selected(); sel == nil || sel.ID != "2" {
		t.Fatalf("cursor should match view order: %+v", sel)
	}
	m.Filter = "zzz-no-match"
	if sel := m.Selected(); sel != nil {
		t.Fatalf("selected on empty filter should be nil")
	}
	m.Filter = ""
	m.MarkRead("2")
	if m.UnreadCount() != 0 {
		t.Fatalf("mark read failed")
	}
	m.UpdateNotes([]*models.Notification{{ID: "3", Title: "x", Unread: true}})
	m.MarkAllRead()
	if m.UnreadCount() != 0 {
		t.Fatalf("mark all failed")
	}
}
