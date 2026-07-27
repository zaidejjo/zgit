// Package models defines shared domain types used by the core engine.
// All presentation layers (TUI, Desktop) depend on these types.
package models

import "time"

// Commit represents a single Git commit.
type Commit struct {
	Hash      string    `json:"hash"`
	Author    string    `json:"author"`
	Email     string    `json:"email"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	RefNames  string    `json:"ref_names,omitempty"` // branches/tags pointing to this commit
}

// Diff represents the delta between two Git refs.
type Diff struct {
	Files        []FileChange `json:"files"`
	TotalAdds    int          `json:"total_additions"`
	TotalDeletes int          `json:"total_deletions"`
}

// FileChangeType indicates the kind of change to a file.
type FileChangeType int

const (
	FileAdded       FileChangeType = iota // A
	FileModified                          // M
	FileDeleted                           // D
	FileRenamed                           // R
	FileCopied                            // C
	FileTypeChanged                       // T
	FileUnmerged                          // U
)

// String returns the short status letter for FileChangeType.
func (t FileChangeType) String() string {
	switch t {
	case FileAdded:
		return "A"
	case FileModified:
		return "M"
	case FileDeleted:
		return "D"
	case FileRenamed:
		return "R"
	case FileCopied:
		return "C"
	case FileTypeChanged:
		return "T"
	case FileUnmerged:
		return "U"
	default:
		return "?"
	}
}

// FileChange describes a single file's change in a diff.
type FileChange struct {
	Type        FileChangeType `json:"type"`
	OldPath     string         `json:"old_path,omitempty"`
	NewPath     string         `json:"new_path,omitempty"`
	Additions   int            `json:"additions"`
	Deletions   int            `json:"deletions"`
	IsBinary    bool           `json:"is_binary"`
	UnifiedDiff string         `json:"unified_diff,omitempty"` // full patch text
}

// Stash represents a single stash entry.
type Stash struct {
	Index   int       `json:"index"`
	Message string    `json:"message"`
	Hash    string    `json:"hash"`
	Time    time.Time `json:"time"`
}
