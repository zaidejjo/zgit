package views

import (
	"strings"
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

func TestBranchSelfTrackingShort(t *testing.T) {
	m := NewBranchModel()
	m.UpdateBranches([]*models.Branch{
		{Name: "dev", IsHead: true, Upstream: "origin/dev", LatestMsg: "Merge PR #61"},
		{Name: "feat/x", Upstream: "origin/feat/x", LatestMsg: "wip"},
		{Name: "other", Upstream: "fork/other", Ahead: 1, LatestMsg: "change"},
	})
	out := m.View(100)
	if strings.Contains(out, "origin/dev") || strings.Contains(out, "origin/feat/x") {
		t.Fatalf("self-tracking upstream should collapse:\n%s", out)
	}
	if !strings.Contains(out, "fork/other") {
		t.Fatalf("foreign upstream should stay:\n%s", out)
	}
	// The branch name owns a fixed column, so the messages line up.
	if strings.Count(out, "Merge PR #61") != 1 || strings.Count(out, "wip") != 1 {
		t.Fatalf("messages missing:\n%s", out)
	}
}

func TestPaletteFuzzyRanking(t *testing.T) {
	m := NewPaletteModel()
	m.Commands = []PaletteCommand{
		{ID: "commit-push", Label: "Commit & Push"},
		{ID: "stash", Label: "Stash"},
		{ID: "commit", Label: "Commit"},
	}
	m.Input = "commit"
	m.filter()
	if len(m.Filtered) != 2 {
		t.Fatalf("filtered = %v", m.Filtered)
	}
	// Exact short prefix "Commit" outranks longer "Commit & Push".
	first := m.Commands[m.Filtered[0]]
	if first.ID != "commit" {
		t.Fatalf("first = %q, want commit", first.ID)
	}
	m.Input = "zzz"
	m.filter()
	if len(m.Filtered) != 0 {
		t.Fatalf("want empty, got %v", m.Filtered)
	}
}

func TestDialogValidators(t *testing.T) {
	if ValidateCommitSubject("") == nil {
		t.Errorf("empty subject should fail")
	}
	if ValidateCommitSubject("   ") == nil {
		t.Errorf("blank subject should fail")
	}
	if ValidateCommitSubject(strings.Repeat("x", 73)) == nil {
		t.Errorf("73-char subject should fail")
	}
	if ValidateCommitSubject("fix: tidy imports") != nil {
		t.Errorf("valid subject should pass")
	}
	if ValidatePRTitle("") == nil {
		t.Errorf("empty title should fail")
	}
	if ValidatePRTitle("Add feature") != nil {
		t.Errorf("valid title should pass")
	}
}

func TestConfigStepCompletion(t *testing.T) {
	m := NewConfigModel()
	m.OpenAIProviderKey("openai")
	if m.State != ConfigInput || m.Field != FieldAIAPIKey {
		t.Fatalf("key step = %+v", m)
	}
	m.inputVal = "sk-test"
	m.completeStep()
	if m.State != ConfigLoading || m.GetInput() != "sk-test" {
		t.Fatalf("after complete = %+v input=%q", m.State, m.GetInput())
	}

	m.OpenAIProviderModel("openai", []string{"gpt-4o"}, []string{"my-model"})
	if m.State != ConfigSelect {
		t.Fatalf("model step should be select, = %v", m.State)
	}
	foundCustom := false
	for _, it := range m.SelectItems {
		if IsCustomModelOption(it) {
			foundCustom = true
		}
	}
	if !foundCustom {
		t.Fatalf("custom sentinel missing: %v", m.SelectItems)
	}
	m.selectVal = "my-model"
	m.completeStep()
	if m.State != ConfigLoading || m.GetSelectedItem() != "my-model" {
		t.Fatalf("after select = %+v item=%q", m.State, m.GetSelectedItem())
	}
	if m.GetSelectedProvider() != "my-model" {
		t.Fatalf("provider passthrough = %q", m.GetSelectedProvider())
	}
}

func TestWorktreeFiltered(t *testing.T) {
	m := NewWorktreeModel()
	m.UpdateWorktrees([]*models.Worktree{
		{Path: "/r/main", Branch: "dev", IsMain: true},
		{Path: "/r/feat", Branch: "feat/x"},
	})
	m.Filter = "feat"
	if got := m.Filtered(); len(got) != 1 || got[0].Branch != "feat/x" {
		t.Fatalf("filtered = %+v", got)
	}
	if out := m.View(100); !strings.Contains(out, "WORKTREES") || !strings.Contains(out, "1/2 · feat") {
		t.Fatalf("title missing:\n%s", out)
	}
}

func TestStashFiltered(t *testing.T) {
	m := NewStashModel()
	m.UpdateStash([]*models.Stash{
		{Index: 0, Message: "wip login"},
		{Index: 1, Message: "experiment"},
	})
	m.Filter = "login"
	if got := m.Filtered(); len(got) != 1 || got[0].Index != 0 {
		t.Fatalf("filtered = %+v", got)
	}
}

func TestRefsSections(t *testing.T) {
	m := NewRefsModel()
	m.UpdateRefs([]string{"v1.0", "v2.0"}, []*models.Remote{
		{Name: "origin", URL: "git@github.com:o/r.git"},
	})
	if m.Section != RefsTags {
		t.Fatalf("default section should be tags")
	}
	m.Toggle()
	if m.Section != RefsRemotes {
		t.Fatalf("toggle should flip section")
	}
	// Filter applies per-section lists; tags list matches one.
	m.Section = RefsTags
	m.Tags.Filter = "v1"
	if got := m.Tags.Filtered(); len(got) != 1 {
		t.Fatalf("tags filtered = %v", got)
	}
	if out := m.View(100); !strings.Contains(out, "origin") {
		t.Fatalf("remotes section missing:\n%s", out)
	}
}

func TestPromptKeys(t *testing.T) {
	var p Prompt
	if done, cancel := p.TypeKey("a"); done || cancel {
		t.Fatalf("closed prompt should ignore keys")
	}
	p.Open("name: ")
	for _, k := range []string{"v", "1", ".", "0"} {
		if done, cancel := p.TypeKey(k); done || cancel {
			t.Fatalf("typing should not finish")
		}
	}
	if p.Value != "v1.0" {
		t.Fatalf("value = %q", p.Value)
	}
	p.TypeKey("backspace")
	if p.Value != "v1." {
		t.Fatalf("after backspace = %q", p.Value)
	}
	p.TypeKey("0")
	if done, _ := p.TypeKey("enter"); !done || !p.Active && p.Value != "v1.0" {
		t.Fatalf("enter should finish")
	}
	p.Open("x: ")
	if _, cancel := p.TypeKey("esc"); !cancel || p.Active {
		t.Fatalf("esc should cancel")
	}
	if !strings.Contains(p.View(), "") {
		t.Fatalf("closed view should be empty, got %q", p.View())
	}
}
