package git

import (
	"testing"
	"time"

	"github.com/zaidejjo/zgit/pkg/core/models"
)

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	ti, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return ti
}

func TestParseBranchInfo(t *testing.T) {
	tests := []struct {
		name string
		data string
		want *branchInfo
	}{
		{
			name: "tracked branch ahead/behind",
			data: "# branch.head main\n# branch.upstream origin/main\n# branch.ab +2 -1\n",
			want: &branchInfo{branch: "main", upstream: "origin/main", ahead: 2, behind: 1},
		},
		{
			name: "detached HEAD",
			data: "# branch.head (detached HEAD)\n# branch.upstream\n# branch.ab +0 -0\n",
			want: &branchInfo{branch: "(detached HEAD)", ahead: 0, behind: 0},
		},
		{
			name: "no upstream",
			data: "# branch.head feature\n# branch.upstream\n# branch.ab +0 -0\n",
			want: &branchInfo{branch: "feature", ahead: 0, behind: 0},
		},
		{
			name: "empty input",
			data: "",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseBranchInfo([]byte(tt.data))
			if tt.want == nil {
				if got != nil {
					t.Errorf("ParseBranchInfo() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("ParseBranchInfo() returned nil, want non-nil")
			}
			if got.branch != tt.want.branch {
				t.Errorf("branch = %q, want %q", got.branch, tt.want.branch)
			}
			if got.upstream != tt.want.upstream {
				t.Errorf("upstream = %q, want %q", got.upstream, tt.want.upstream)
			}
			if got.ahead != tt.want.ahead {
				t.Errorf("ahead = %d, want %d", got.ahead, tt.want.ahead)
			}
			if got.behind != tt.want.behind {
				t.Errorf("behind = %d, want %d", got.behind, tt.want.behind)
			}
		})
	}
}

func TestParsePorcelainV2(t *testing.T) {
	tests := []struct {
		name string
		data string
		bi   *branchInfo
		want *models.Status
	}{
		{
			name: "clean working tree",
			data: "",
			bi:   &branchInfo{branch: "main", upstream: "origin/main"},
			want: &models.Status{
				Branch:   "main",
				Upstream: "origin/main",
				Files:    []models.FileStatus{},
				IsClean:  true,
			},
		},
		{
			name: "modified staged and unstaged",
			data: "1 .M N... 100644 100644 100644 0000000000 1111111111 file1.txt\n1 M. N... 100644 100644 100644 1111111111 2222222222 file2.txt\n",
			bi:   &branchInfo{branch: "main"},
			want: &models.Status{
				Branch:  "main",
				IsClean: false,
				Files: []models.FileStatus{
					{Path: "file1.txt", Staged: models.StatusUnmodified, Unstaged: models.StatusModified},
					{Path: "file2.txt", Staged: models.StatusModified, Unstaged: models.StatusUnmodified},
				},
			},
		},
		{
			name: "untracked file",
			data: "? untracked.go\n",
			bi:   &branchInfo{branch: "main"},
			want: &models.Status{
				Branch:  "main",
				IsClean: false,
				Files: []models.FileStatus{
					{Path: "untracked.go", Staged: models.StatusUntracked, Unstaged: models.StatusUnmodified},
				},
			},
		},
		{
			name: "renamed file",
			data: "2 R. N... 100644 100644 100644 0000000000 1111111111 R100 old_name.go\tnew_name.go\n",
			bi:   &branchInfo{branch: "main"},
			want: &models.Status{
				Branch:  "main",
				IsClean: false,
				Files: []models.FileStatus{
					{Path: "new_name.go", OldPath: "old_name.go", Staged: models.StatusRenamed, Unstaged: models.StatusUnmodified},
				},
			},
		},
		{
			name: "deleted file",
			data: "1 D. N... 100644 100644 100644 0000000000 1111111111 deleted.txt\n",
			bi:   &branchInfo{branch: "main"},
			want: &models.Status{
				Branch:  "main",
				IsClean: false,
				Files: []models.FileStatus{
					{Path: "deleted.txt", Staged: models.StatusDeleted, Unstaged: models.StatusUnmodified},
				},
			},
		},
		{
			name: "filename with spaces",
			data: "1 .M N... 100644 100644 100644 0000000000 1111111111 my file with spaces.txt\n",
			bi:   &branchInfo{branch: "main"},
			want: &models.Status{
				Branch:  "main",
				IsClean: false,
				Files: []models.FileStatus{
					{Path: "my file with spaces.txt", Staged: models.StatusUnmodified, Unstaged: models.StatusModified},
				},
			},
		},
		{
			name: "mixed changes",
			data: "1 .M N... 100644 100644 100644 0000000000 1111111111 working.txt\n1 M. N... 100644 100644 100644 1111111111 2222222222 staged.txt\n1 MM N... 100644 100644 100644 3333333333 4444444444 both.txt\n? new.go\n",
			bi:   &branchInfo{branch: "dev", upstream: "origin/dev", ahead: 3, behind: 1},
			want: &models.Status{
				Branch:   "dev",
				Upstream: "origin/dev",
				Ahead:    3,
				Behind:   1,
				IsClean:  false,
				Files: []models.FileStatus{
					{Path: "working.txt", Staged: models.StatusUnmodified, Unstaged: models.StatusModified},
					{Path: "staged.txt", Staged: models.StatusModified, Unstaged: models.StatusUnmodified},
					{Path: "both.txt", Staged: models.StatusModified, Unstaged: models.StatusModified},
					{Path: "new.go", Staged: models.StatusUntracked, Unstaged: models.StatusUnmodified},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePorcelainV2([]byte(tt.data), tt.bi)
			if got.Branch != tt.want.Branch {
				t.Errorf("Branch = %q, want %q", got.Branch, tt.want.Branch)
			}
			if got.Upstream != tt.want.Upstream {
				t.Errorf("Upstream = %q, want %q", got.Upstream, tt.want.Upstream)
			}
			if got.IsClean != tt.want.IsClean {
				t.Errorf("IsClean = %v, want %v", got.IsClean, tt.want.IsClean)
			}
			if len(got.Files) != len(tt.want.Files) {
				t.Fatalf("len(Files) = %d, want %d\ngot: %+v", len(got.Files), len(tt.want.Files), got.Files)
			}
			for i, f := range got.Files {
				wf := tt.want.Files[i]
				if f.Path != wf.Path {
					t.Errorf("Files[%d].Path = %q, want %q", i, f.Path, wf.Path)
				}
				if f.Staged != wf.Staged {
					t.Errorf("Files[%d].Staged = %v, want %v", i, f.Staged, wf.Staged)
				}
				if f.Unstaged != wf.Unstaged {
					t.Errorf("Files[%d].Unstaged = %v, want %v", i, f.Unstaged, wf.Unstaged)
				}
			}
		})
	}
}

func TestParseLog(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []*models.Commit
	}{
		{
			name: "single commit",
			data: "abc123\t\tJohn Doe\tjohn@ex.com\t1700000000\tInitial commit\t",
			want: []*models.Commit{
				{
					Hash:      "abc123",
					Parents:   nil,
					Author:    "John Doe",
					Email:     "john@ex.com",
					Timestamp: time.Unix(1700000000, 0),
					Message:   "Initial commit",
				},
			},
		},
		{
			name: "multiple commits",
			data: "aaa\tp001\tAlice\talice@ex.com\t1700000100\tFirst commit\t(tag: v1.0)\nbbb\tp002 p003\tBob\tbob@ex.com\t1700000200\tSecond commit\t(main)\n",
			want: []*models.Commit{
				{
					Hash: "aaa", Parents: []string{"p001"}, Author: "Alice", Email: "alice@ex.com",
					Timestamp: time.Unix(1700000100, 0), Message: "First commit", RefNames: "(tag: v1.0)",
				},
				{
					Hash: "bbb", Parents: []string{"p002", "p003"}, Author: "Bob", Email: "bob@ex.com",
					Timestamp: time.Unix(1700000200, 0), Message: "Second commit", RefNames: "(main)",
				},
			},
		},
		{
			name: "empty output",
			data: "",
			want: []*models.Commit{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLog([]byte(tt.data))
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i, c := range got {
				wc := tt.want[i]
				if c.Hash != wc.Hash {
					t.Errorf("Commit[%d].Hash = %q, want %q", i, c.Hash, wc.Hash)
				}
				if c.Author != wc.Author {
					t.Errorf("Commit[%d].Author = %q, want %q", i, c.Author, wc.Author)
				}
				if c.Message != wc.Message {
					t.Errorf("Commit[%d].Message = %q, want %q", i, c.Message, wc.Message)
				}
				if !c.Timestamp.Equal(wc.Timestamp) {
					t.Errorf("Commit[%d].Timestamp = %v, want %v", i, c.Timestamp, wc.Timestamp)
				}
				if len(c.Parents) != len(wc.Parents) {
					t.Errorf("Commit[%d].Parents len = %d, want %d", i, len(c.Parents), len(wc.Parents))
				} else {
					for j, p := range c.Parents {
						if p != wc.Parents[j] {
							t.Errorf("Commit[%d].Parents[%d] = %q, want %q", i, j, p, wc.Parents[j])
						}
					}
				}
			}
		})
	}
}

func TestParseBranches(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []*models.Branch
	}{
		{
			name: "local branches",
			data: "main\t \torigin/main\t\tabc123\tLatest commit\nfeature\t*\t\t\tdef456\tFeature work\n",
			want: []*models.Branch{
				{
					Name: "main", IsHead: false, Type: models.LocalBranch,
					Upstream: "origin/main", Ahead: 0, Behind: 0,
					FullRef: "refs/heads/main", LatestHash: "abc123", LatestMsg: "Latest commit",
				},
				{
					Name: "feature", IsHead: true, Type: models.LocalBranch,
					Upstream: "", Ahead: 0, Behind: 0,
					FullRef: "refs/heads/feature", LatestHash: "def456", LatestMsg: "Feature work",
				},
			},
		},
		{
			name: "branches with ahead/behind",
			data: "main\t \torigin/main\tahead 3, behind 1\tabc123\tLatest\n",
			want: []*models.Branch{
				{
					Name: "main", IsHead: false, Type: models.LocalBranch,
					Upstream: "origin/main", Ahead: 3, Behind: 1,
					FullRef: "refs/heads/main", LatestHash: "abc123", LatestMsg: "Latest",
				},
			},
		},
		{
			name: "malformed line",
			data: "incomplete\n",
			want: nil, // Expect error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBranches([]byte(tt.data))
			if tt.want == nil {
				if err == nil {
					t.Error("ParseBranches() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i, b := range got {
				wb := tt.want[i]
				if b.Name != wb.Name {
					t.Errorf("Branch[%d].Name = %q, want %q", i, b.Name, wb.Name)
				}
				if b.IsHead != wb.IsHead {
					t.Errorf("Branch[%d].IsHead = %v, want %v", i, b.IsHead, wb.IsHead)
				}
				if b.Upstream != wb.Upstream {
					t.Errorf("Branch[%d].Upstream = %q, want %q", i, b.Upstream, wb.Upstream)
				}
			}
		})
	}
}

func TestParseDiffNumstat(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []models.FileChange
	}{
		{
			name: "modified files",
			data: "10\t5\tsrc/main.go\n3\t1\tsrc/utils.go\n",
			want: []models.FileChange{
				{NewPath: "src/main.go", Additions: 10, Deletions: 5, Type: models.FileModified},
				{NewPath: "src/utils.go", Additions: 3, Deletions: 1, Type: models.FileModified},
			},
		},
		{
			name: "binary file",
			data: "-\t-\timage.png\n",
			want: []models.FileChange{
				{NewPath: "image.png", IsBinary: true, Type: models.FileModified},
			},
		},
		{
			name: "renamed file",
			data: "0\t0\told.txt\tnew.txt\n",
			want: []models.FileChange{
				{OldPath: "old.txt", NewPath: "new.txt", Additions: 0, Deletions: 0, Type: models.FileRenamed},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDiffNumstat([]byte(tt.data))
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i, f := range got {
				wf := tt.want[i]
				if f.NewPath != wf.NewPath {
					t.Errorf("File[%d].NewPath = %q, want %q", i, f.NewPath, wf.NewPath)
				}
				if f.IsBinary != wf.IsBinary {
					t.Errorf("File[%d].IsBinary = %v, want %v", i, f.IsBinary, wf.IsBinary)
				}
				if f.Type != wf.Type {
					t.Errorf("File[%d].Type = %v, want %v", i, f.Type, wf.Type)
				}
			}
		})
	}
}

func TestParseRemotes(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []*models.Remote
	}{
		{
			name: "single remote",
			data: "origin\tgit@github.com:user/repo.git (fetch)\norigin\tgit@github.com:user/repo.git (push)\n",
			want: []*models.Remote{
				{Name: "origin", URL: "git@github.com:user/repo.git", Type: "fetch"},
				{Name: "origin", URL: "git@github.com:user/repo.git", Type: "push"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRemotes([]byte(tt.data))
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i, r := range got {
				wr := tt.want[i]
				if r.Name != wr.Name {
					t.Errorf("Remote[%d].Name = %q, want %q", i, r.Name, wr.Name)
				}
				if r.URL != wr.URL {
					t.Errorf("Remote[%d].URL = %q, want %q", i, r.URL, wr.URL)
				}
				if r.Type != wr.Type {
					t.Errorf("Remote[%d].Type = %q, want %q", i, r.Type, wr.Type)
				}
			}
		})
	}
}

func TestParseStashList(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []*models.Stash
	}{
		{
			name: "multiple stashes",
			data: "abc123def\tOn main: fix formatting\ndef456ghi\tOn feature: WIP refactor\n",
			want: []*models.Stash{
				{Index: 0, Hash: "abc123def", Message: "On main: fix formatting"},
				{Index: 1, Hash: "def456ghi", Message: "On feature: WIP refactor"},
			},
		},
		{
			name: "empty",
			data: "",
			want: []*models.Stash{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStashList([]byte(tt.data))
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i, s := range got {
				ws := tt.want[i]
				if s.Message != ws.Message {
					t.Errorf("Stash[%d].Message = %q, want %q", i, s.Message, ws.Message)
				}
			}
		})
	}
}

func TestParseRevList(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []string
	}{
		{
			name: "multiple hashes",
			data: "abc123\ndef456\nghi789\n",
			want: []string{"abc123", "def456", "ghi789"},
		},
		{
			name: "empty",
			data: "",
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRevList([]byte(tt.data))
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i, h := range got {
				if h != tt.want[i] {
					t.Errorf("rev[%d] = %q, want %q", i, h, tt.want[i])
				}
			}
		})
	}
}

// TestStatusHelperMethods tests the convenience methods on Status.
func TestStatusHelperMethods(t *testing.T) {
	s := &models.Status{
		Files: []models.FileStatus{
			{Path: "staged.txt", Staged: models.StatusAdded, Unstaged: models.StatusUnmodified},
			{Path: "unstaged.txt", Staged: models.StatusUnmodified, Unstaged: models.StatusModified},
			{Path: "both.txt", Staged: models.StatusModified, Unstaged: models.StatusModified},
			{Path: "new.go", Staged: models.StatusUntracked, Unstaged: models.StatusUnmodified},
			{Path: "unchanged.go", Staged: models.StatusUnmodified, Unstaged: models.StatusUnmodified},
		},
	}

	staged := s.StagedFiles()
	if len(staged) != 2 {
		t.Errorf("StagedFiles() len = %d, want 2", len(staged))
	}

	unstaged := s.UnstagedFiles()
	if len(unstaged) != 2 {
		t.Errorf("UnstagedFiles() len = %d, want 2 (unstaged.txt + both.txt)", len(unstaged))
	}

	untracked := s.UntrackedFiles()
	if len(untracked) != 1 {
		t.Errorf("UntrackedFiles() len = %d, want 1", len(untracked))
	}
	if untracked[0].Path != "new.go" {
		t.Errorf("UntrackedFiles()[0].Path = %q, want %q", untracked[0].Path, "new.go")
	}
}

func TestCommitRefNames(t *testing.T) {
	data := "abc123\tparentHash\tAlice\talice@ex.com\t1700000100\tFix bug\t(tag: v1.0, main)\n"
	got, err := ParseLog([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].RefNames != "(tag: v1.0, main)" {
		t.Errorf("RefNames = %q, want %q", got[0].RefNames, "(tag: v1.0, main)")
	}
}
