package models

// RebaseAction defines the action for a single commit during interactive rebase.
type RebaseAction string

const (
	RebasePick   RebaseAction = "pick"
	RebaseReword RebaseAction = "reword"
	RebaseSquash RebaseAction = "squash"
	RebaseFixup  RebaseAction = "fixup"
	RebaseDrop   RebaseAction = "drop"
)

// RebaseCommitOp describes what to do with a single commit during rebase.
type RebaseCommitOp struct {
	SHA        string       `json:"sha"`
	Action     RebaseAction `json:"action"`
	NewMessage string       `json:"new_message,omitempty"` // for reword
}

// RebaseSequenceOptions describes the full rebase operation.
type RebaseSequenceOptions struct {
	Onto    string           `json:"onto"`    // base ref to rebase onto
	Commits []RebaseCommitOp `json:"commits"` // ordered list of commit operations
}

// RebaseResult describes the outcome of a rebase.
type RebaseResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}
