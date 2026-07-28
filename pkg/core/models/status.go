package models

// StatusType encodes the working-tree status of a file (XY code from --porcelain).
type StatusType int

const (
	StatusUntracked          StatusType = iota // ?
	StatusAdded                                // A
	StatusModified                             // M
	StatusDeleted                              // D
	StatusRenamed                              // R
	StatusCopied                               // C
	StatusUpdatedButUnmerged                   // U
	StatusUnmodified                           // (space)
	StatusIgnored                              // !
)

// String returns the short status letter.
func (s StatusType) String() string {
	switch s {
	case StatusUntracked:
		return "?"
	case StatusAdded:
		return "A"
	case StatusModified:
		return "M"
	case StatusDeleted:
		return "D"
	case StatusRenamed:
		return "R"
	case StatusCopied:
		return "C"
	case StatusUpdatedButUnmerged:
		return "U"
	case StatusUnmodified:
		return " "
	case StatusIgnored:
		return "!"
	default:
		return "?"
	}
}

// FileStatus holds the staged and unstaged status for a single file.
type FileStatus struct {
	Path     string     `json:"path"`
	OldPath  string     `json:"old_path,omitempty"` // for renames
	Staged   StatusType `json:"staged"`             // X in porcelain XY
	Unstaged StatusType `json:"unstaged"`           // Y in porcelain XY
}

// Status represents the full working-tree status.
type Status struct {
	Branch         string       `json:"branch"`
	Upstream       string       `json:"upstream,omitempty"`
	Ahead          int          `json:"ahead"`
	Behind         int          `json:"behind"`
	Files          []FileStatus `json:"files"`
	StagedCount    int          `json:"staged_count"`
	UnstagedCount  int          `json:"unstaged_count"`
	UntrackedCount int          `json:"untracked_count"`
	IsClean        bool         `json:"is_clean"`
	IsMerging      bool         `json:"is_merging"`
	IsRebasing     bool         `json:"is_rebasing"`
	IsCherryPick   bool         `json:"is_cherry_pick"`
	IsReverting    bool         `json:"is_reverting"`
	IsBisecting    bool         `json:"is_bisecting"`
}

// StagedFiles returns only files with staged changes.
func (s *Status) StagedFiles() []FileStatus {
	var out []FileStatus
	for _, f := range s.Files {
		if f.Staged != StatusUnmodified && f.Staged != StatusUntracked {
			out = append(out, f)
		}
	}
	return out
}

// UnstagedFiles returns only files with unstaged (working-tree) changes.
func (s *Status) UnstagedFiles() []FileStatus {
	var out []FileStatus
	for _, f := range s.Files {
		if f.Unstaged != StatusUnmodified {
			out = append(out, f)
		}
	}
	return out
}

// UntrackedFiles returns only untracked files.
func (s *Status) UntrackedFiles() []FileStatus {
	var out []FileStatus
	for _, f := range s.Files {
		if f.Staged == StatusUntracked {
			out = append(out, f)
		}
	}
	return out
}

// OptimisticStage marks a file as staged in the local state (instant feedback).
// Call after git Add succeeds but before Refresh() response arrives.
func (s *Status) OptimisticStage(path string) {
	for i := range s.Files {
		if s.Files[i].Path != path {
			continue
		}
		f := &s.Files[i]
		// If untracked → now added
		if f.Staged == StatusUntracked {
			f.Staged = StatusAdded
		}
		// If unstaged changes → move to staged
		if f.Unstaged != StatusUnmodified && f.Unstaged != StatusUntracked {
			f.Staged = f.Unstaged
			f.Unstaged = StatusUnmodified
		}
		break
	}
	s.Recount()
}

// OptimisticUnstage marks a file as unstaged in the local state.
func (s *Status) OptimisticUnstage(path string) {
	for i := range s.Files {
		if s.Files[i].Path != path {
			continue
		}
		f := &s.Files[i]
		if f.Staged != StatusUnmodified && f.Staged != StatusUntracked {
			f.Unstaged = f.Staged
			f.Staged = StatusUnmodified
		}
		break
	}
	s.Recount()
}

// OptimisticStageAll stages all unstaged and untracked files locally.
func (s *Status) OptimisticStageAll() {
	for i := range s.Files {
		f := &s.Files[i]
		if f.Staged == StatusUntracked {
			f.Staged = StatusAdded
		} else if f.Unstaged != StatusUnmodified && f.Unstaged != StatusUntracked {
			f.Staged = f.Unstaged
			f.Unstaged = StatusUnmodified
		}
	}
	s.Recount()
}

// OptimisticUnstageAll unstages all staged files locally.
func (s *Status) OptimisticUnstageAll() {
	for i := range s.Files {
		f := &s.Files[i]
		if f.Staged != StatusUnmodified && f.Staged != StatusUntracked {
			f.Unstaged = f.Staged
			f.Staged = StatusUnmodified
		}
	}
	s.Recount()
}

// Recount recalculates StagedCount, UnstagedCount, UntrackedCount, IsClean.
func (s *Status) Recount() {
	staged, unstaged, untracked := 0, 0, 0
	for _, f := range s.Files {
		if f.Staged != StatusUnmodified && f.Staged != StatusUntracked {
			staged++
		}
		if f.Unstaged != StatusUnmodified {
			unstaged++
		}
		if f.Staged == StatusUntracked {
			untracked++
		}
	}
	s.StagedCount = staged
	s.UnstagedCount = unstaged
	s.UntrackedCount = untracked
	s.IsClean = len(s.Files) == 0 || (staged == 0 && unstaged == 0 && untracked == 0)
}
