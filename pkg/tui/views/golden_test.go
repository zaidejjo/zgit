package views

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/zaidejjo/zgit/pkg/core/models"
)

var goldenANSI = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

// assertGolden compares plain (ANSI-stripped) view output to testdata.
// Regenerate with UPDATE_GOLDEN=1 after deliberate visual changes.
func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	plain := goldenANSI.ReplaceAllString(got, "")
	path := filepath.Join("testdata", name+".golden")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(plain), 0644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden %s missing (run with UPDATE_GOLDEN=1): %v", path, err)
	}
	if plain != string(want) {
		t.Fatalf("golden %s mismatch:\n--- got ---\n%s\n--- want ---\n%s", name, plain, want)
	}
}

func TestGoldenDashboard(t *testing.T) {
	m := NewDashModel()
	m.Height = 30
	m.UpdateEntries([]DashEntry{
		{Name: "zgit", Path: "/r/zgit", Branch: "dev", Clean: true, Favorite: true, Current: true},
		{Name: "zz_lang", Path: "/r/zz_lang", Branch: "main", Clean: false, Dirty: 3, Ahead: 1},
		{Name: "gone", Path: "/r/gone", Err: "unreachable"},
	})
	assertGolden(t, "dashboard", m.View(100))
}

func TestGoldenNotes(t *testing.T) {
	m := NewNotesModel()
	m.Height = 30
	m.UpdateNotes([]*models.Notification{
		{ID: "1", Title: "Review requested", Type: "PullRequest", Reason: "review_requested", RepoName: "o/r", Unread: true},
		{ID: "2", Title: "Mentioned you", Type: "Issue", Reason: "mention", RepoName: "o/r"},
	})
	assertGolden(t, "notes", m.View(100))
}

func TestGoldenActions(t *testing.T) {
	m := NewActionsModel()
	m.Height = 30
	m.UpdateRuns([]*models.WorkflowRun{
		{ID: 1, WorkflowName: "CI", Branch: "dev", HeadSHA: "abc1234def", RunNumber: 10, Status: "COMPLETED", Conclusion: "SUCCESS"},
		{ID: 2, WorkflowName: "Docs", Branch: "main", HeadSHA: "def5678", RunNumber: 3, Status: "IN_PROGRESS"},
	})
	assertGolden(t, "actions", m.View(100))
}
