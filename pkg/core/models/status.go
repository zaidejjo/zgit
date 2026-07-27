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
