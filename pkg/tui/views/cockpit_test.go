package views

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
)

// cockpitRows builds a commit list with every decoration the log view
// has to draw.
func cockpitCommits() []*models.Commit {
	now := time.Now()
	return []*models.Commit{
		{Hash: "abc1234567", Message: "feat(loader): dispatch on 500 methods", RefNames: "HEAD -> dev, origin/dev, tag: v0.3.0", Timestamp: now.Add(-2 * time.Minute)},
		{Hash: "def5678901", Message: "fix: an unusually long subject that has to give way to the columns around it", RefNames: "origin/feat/x", Timestamp: now.Add(-72 * time.Hour)},
		{Hash: "0123456789", Message: "chore: no refs at all", Timestamp: now},
	}
}

// TestLogRowsFitEveryWidth is the width law applied to the commit grid:
// the row is a table, and a table that overflows wraps the pane.
func TestLogRowsFitEveryWidth(t *testing.T) {
	for width := 20; width <= 200; width += 3 {
		for _, c := range cockpitCommits() {
			row := commitRow(c, "│ ● ", width)
			if w := lipgloss.Width(row); w > width {
				t.Fatalf("width %d: row is %d cells: %q", width, w, goldenANSI.ReplaceAllString(row, ""))
			}
		}
	}
}

// TestLogRowKeepsSubjectWhenNarrow: refs are decoration, the subject is
// the reason the row exists.
func TestLogRowKeepsSubjectWhenNarrow(t *testing.T) {
	c := cockpitCommits()[0]

	narrow := goldenANSI.ReplaceAllString(commitRow(c, "● ", 40), "")
	if !strings.Contains(narrow, "feat(loader)") {
		t.Fatalf("subject lost at width 40: %q", narrow)
	}
	if strings.Contains(narrow, "v0.3.0") {
		t.Fatalf("refs should give way first at width 40: %q", narrow)
	}

	wide := goldenANSI.ReplaceAllString(commitRow(c, "● ", 140), "")
	for _, want := range []string{"abc1234", "dev", "v0.3.0", "feat(loader)"} {
		if !strings.Contains(wide, want) {
			t.Fatalf("width 140: missing %q in %q", want, wide)
		}
	}
	// Columns: hash first, then the age, then the subject.
	if i, j := strings.Index(wide, "abc1234"), strings.Index(wide, "feat(loader)"); i < 0 || j < 0 || i > j {
		t.Fatalf("hash must precede the subject: %q", wide)
	}
}

// TestLogRowAgeColumnIsFixed: rows must not shift when the age changes
// width ("now" vs "12mo").
func TestLogRowAgeColumnIsFixed(t *testing.T) {
	now := time.Now()
	commits := []*models.Commit{
		{Hash: "aaa1111", Message: "same", Timestamp: now},
		{Hash: "bbb2222", Message: "same", Timestamp: now.Add(-400 * 24 * time.Hour)},
	}
	cols := make([]int, len(commits))
	for i, c := range commits {
		row := goldenANSI.ReplaceAllString(commitRow(c, "● ", 120), "")
		cols[i] = strings.Index(row, "same")
	}
	if cols[0] != cols[1] {
		t.Fatalf("subject column moved with the age: %v", cols)
	}
}

// TestBranchRowAlignsMessages pins the fixed name column: this is the
// "name and message fused into one blob" defect.
func TestBranchRowAlignsMessages(t *testing.T) {
	short := goldenANSI.ReplaceAllString(branchRow(&models.Branch{Name: "dev", IsHead: true, LatestMsg: "msg one"}, 100), "")
	long := goldenANSI.ReplaceAllString(branchRow(&models.Branch{Name: "feature/a-rather-long-branch", LatestMsg: "msg two"}, 100), "")
	if a, b := strings.Index(short, "msg one"), strings.Index(long, "msg two"); a != b {
		t.Fatalf("messages not aligned: %d vs %d\n%q\n%q", a, b, short, long)
	}
	if strings.Contains(short, "devmsg") || strings.Contains(long, "featmsg") {
		t.Fatalf("name and message fused:\n%q\n%q", short, long)
	}
}

func TestBranchRowFitsEveryWidth(t *testing.T) {
	br := &models.Branch{Name: "feature/500-method-dispatch", Upstream: "origin/feature/500-method-dispatch", Ahead: 3, Behind: 12, LatestMsg: strings.Repeat("long subject ", 8)}
	for width := 12; width <= 200; width += 3 {
		row := branchRow(br, width)
		if w := lipgloss.Width(row); w > width {
			t.Fatalf("width %d: row is %d cells", width, w)
		}
	}
}

// TestStatusFileRowShowsGitCode: a file row states what changed (M/A/D),
// not just that something did.
func TestStatusFileRowShowsGitCode(t *testing.T) {
	f := models.FileStatus{Path: "internal/cli/log.go", Unstaged: models.StatusModified}
	row := goldenANSI.ReplaceAllString(fileRow(f, "M", 80), "")
	if !strings.Contains(row, "M") || !strings.Contains(row, "log.go") || !strings.Contains(row, "internal/cli/") {
		t.Fatalf("row = %q", row)
	}
	// The code owns the first column, so every row starts at the same x.
	nameAt := strings.Index(row, "log.go")
	other := goldenANSI.ReplaceAllString(fileRow(models.FileStatus{Path: "a.go", Unstaged: models.StatusAdded}, "A", 80), "")
	if strings.Index(other, "a.go") != nameAt {
		t.Fatalf("filename column moved: %d vs %d\n%q\n%q", nameAt, strings.Index(other, "a.go"), row, other)
	}
}

// TestStatusFileRowFitsEveryWidth keeps the file table inside its pane,
// including when the directory is long enough to need a left cut.
func TestStatusFileRowFitsEveryWidth(t *testing.T) {
	f := models.FileStatus{
		Path:     "desktop/frontend/src/components/some/deeply/nested/directory/File.go",
		OldPath:  "desktop/old/File.go",
		Unstaged: models.StatusRenamed,
	}
	for width := 6; width <= 200; width += 3 {
		row := fileRow(f, "R", width)
		if w := lipgloss.Width(row); w > width {
			t.Fatalf("width %d: row is %d cells: %q", width, w, goldenANSI.ReplaceAllString(row, ""))
		}
	}
}

// TestStatusCleanStateIsCentered covers the empty state: it is stated in
// the middle of the pane instead of as a top-left line.
func TestStatusCleanStateIsCentered(t *testing.T) {
	width := 60
	out := cleanState(width)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("clean state = %d lines, want 2", len(lines))
	}
	for i, line := range lines {
		plain := goldenANSI.ReplaceAllString(line, "")
		if w := lipgloss.Width(plain); w > width {
			t.Fatalf("line %d is %d cells: %q", i, w, plain)
		}
		lead := len(plain) - len(strings.TrimLeft(plain, " "))
		if lead < 5 {
			t.Fatalf("line %d is not centered: %q", i, plain)
		}
	}
}

// TestBranchLineSeparatesUpstream is the "devorigin/dev" regression: the
// branch and its upstream are distinct cells on the row.
func TestBranchLineSeparatesUpstream(t *testing.T) {
	// A foreign upstream: a self-tracking one (origin/<branch>) is
	// deliberately collapsed, so it cannot express this bug.
	s := &models.Status{Branch: "dev", Upstream: "origin/main", Ahead: 1, Behind: 2}
	plain := goldenANSI.ReplaceAllString(branchLine(s, 80), "")
	if strings.Contains(plain, "devorigin") {
		t.Fatalf("branch and upstream fused: %q", plain)
	}
	if !strings.Contains(plain, "dev") || !strings.Contains(plain, "origin/main") {
		t.Fatalf("missing parts: %q", plain)
	}
	if !strings.Contains(plain, "↑1") || !strings.Contains(plain, "↓2") {
		t.Fatalf("divergence missing: %q", plain)
	}

	// A self-tracking upstream is redundant: the branch chip said it.
	self := goldenANSI.ReplaceAllString(branchLine(&models.Status{Branch: "dev", Upstream: "origin/dev"}, 80), "")
	if strings.Contains(self, "origin/dev") {
		t.Fatalf("self-tracking upstream should collapse: %q", self)
	}

	// And the row obeys the pane at any width.
	for _, width := range []int{4, 8, 12, 20, 40, 120} {
		for _, st := range []*models.Status{s, {Branch: "feature/some-very-long-branch-name", Upstream: "upstream/other", Ahead: 12, Behind: 34, IsRebasing: true}} {
			if w := lipgloss.Width(branchLine(st, width)); w > width {
				t.Fatalf("width %d: branch line is %d cells", width, w)
			}
		}
	}
}

// TestStatusViewFitsPane renders the whole status pane at awkward widths.
func TestStatusViewFitsPane(t *testing.T) {
	m := NewStatusModel()
	m.UpdateStatus(&models.Status{
		Branch:         "feature/500-method-dispatch",
		Upstream:       "origin/feature/500-method-dispatch",
		Ahead:          2,
		Behind:         1,
		StagedCount:    1,
		UnstagedCount:  1,
		UntrackedCount: 1,
		Files: []models.FileStatus{
			{Path: "pkg/core/parser.go", Staged: models.StatusModified},
			{Path: "cmd/zgit/main.go", Unstaged: models.StatusModified},
			{Path: "internal/cli/notes.go", Unstaged: models.StatusUntracked},
		},
	})
	for width := 16; width <= 200; width += 7 {
		out := m.View(width)
		for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Fatalf("width %d: line is %d cells: %q", width, w, goldenANSI.ReplaceAllString(line, ""))
			}
		}
	}
}
