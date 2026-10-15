package views

import (
	"testing"

	"github.com/zaidejjo/zgit/pkg/core/models"
)

func TestPRFiltered(t *testing.T) {
	m := NewPRListModel()
	m.UpdatePRs([]*models.PullRequestSummary{
		{Number: 1, Title: "Fix login bug", Author: "alice", HeadRef: "fix/login"},
		{Number: 2, Title: "Add dashboard", Author: "bob", HeadRef: "feat/dash"},
	})
	if len(m.Filtered()) != 2 {
		t.Fatalf("unfiltered len = %d", len(m.Filtered()))
	}
	m.Filter = "dash"
	got := m.Filtered()
	if len(got) != 1 || got[0].Number != 2 {
		t.Fatalf("filtered = %v", got)
	}
	m.Filter = "ALICE"
	if len(m.Filtered()) != 1 {
		t.Fatalf("case-insensitive failed: %v", m.Filtered())
	}
	m.Filter = "zzz"
	if len(m.Filtered()) != 0 {
		t.Fatalf("want empty, got %v", m.Filtered())
	}
	m.Cursor = 5
	m.ClampCursor()
	if m.Cursor != 0 {
		t.Fatalf("clamp = %d", m.Cursor)
	}
	if m.SelectedPR() != nil {
		t.Fatalf("selected on empty filter should be nil")
	}
}
