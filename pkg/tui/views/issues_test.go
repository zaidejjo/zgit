package views

import (
	"testing"

	"github.com/zaidejjo/zgit/pkg/core/models"
)

func TestIssueFiltered(t *testing.T) {
	m := NewIssueListModel()
	m.UpdateIssues([]*models.Issue{
		{Number: 1, Title: "Crash on startup", Author: "alice"},
		{Number: 2, Title: "Dark mode", Author: "bob"},
	})
	m.Filter = "crash"
	got := m.Filtered()
	if len(got) != 1 || got[0].Number != 1 {
		t.Fatalf("filtered = %v", got)
	}
	m.Filter = "2"
	if len(m.Filtered()) != 1 {
		t.Fatalf("number search failed")
	}
	m.Cursor = 9
	m.ClampCursor()
	if m.Cursor != 0 {
		t.Fatalf("clamp = %d", m.Cursor)
	}
}
