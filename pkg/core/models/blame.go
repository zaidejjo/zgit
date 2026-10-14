package models

import "time"

// BlameLine attributes one line of a file to the commit that last touched it.
type BlameLine struct {
	// LineNo is the 1-based final line number in the blamed revision.
	LineNo int `json:"line_no"`
	// OrigLineNo is the 1-based line number in the originating commit.
	OrigLineNo int `json:"orig_line_no"`
	// Hash is the full commit SHA that last modified the line.
	Hash string `json:"hash"`
	// Author is the commit author name.
	Author string `json:"author"`
	// Email is the commit author email.
	Email string `json:"email"`
	// Timestamp is the author date of the commit.
	Timestamp time.Time `json:"timestamp"`
	// Summary is the commit subject line.
	Summary string `json:"summary"`
	// Content is the source line text (without trailing newline).
	Content string `json:"content"`
	// IsUncommitted is true for lines not yet committed (all-zero SHA).
	IsUncommitted bool `json:"is_uncommitted"`
}

// BlameOptions controls `git blame` behavior.
type BlameOptions struct {
	// Ref is the revision to blame (empty = working tree / HEAD).
	Ref string `json:"ref"`
	// Lines restricts output to a range like "10,20" (-L).
	Lines string `json:"lines,omitempty"`
}
