package models

import "time"

// ReflogAction categorises a reflog entry by the type of Git operation.
type ReflogAction string

const (
	ReflogCommit     ReflogAction = "commit"
	ReflogReset      ReflogAction = "reset"
	ReflogCheckout   ReflogAction = "checkout"
	ReflogMerge      ReflogAction = "merge"
	ReflogRebase     ReflogAction = "rebase"
	ReflogCherryPick ReflogAction = "cherry-pick"
	ReflogRevert     ReflogAction = "revert"
	ReflogBranch     ReflogAction = "branch"
	ReflogAmend      ReflogAction = "amend"
	ReflogUnknown    ReflogAction = "unknown"
)

// ReflogEntry represents a single entry from `git reflog`.
type ReflogEntry struct {
	Sequence  int          `json:"sequence"`           // reflog sequence (HEAD@{N})
	Hash      string       `json:"hash"`               // commit hash after the action
	Action    ReflogAction `json:"action"`             // categorised action type
	Subject   string       `json:"subject"`            // reflog subject (e.g. "commit: fix typo")
	Timestamp time.Time    `json:"timestamp"`          // when the action occurred
	OldHash   string       `json:"old_hash,omitempty"` // previous hash (for resets/checkouts)
	Undoable  bool         `json:"undoable"`           // whether this action can be undone
}

// ReflogList is a list of reflog entries.
type ReflogList []ReflogEntry

// UndoableActions are reflog action types that support automatic undo.
var UndoableActions = map[ReflogAction]bool{
	ReflogCommit:     true,
	ReflogReset:      true,
	ReflogMerge:      true,
	ReflogCherryPick: true,
	ReflogRevert:     true,
	ReflogBranch:     true,
	ReflogAmend:      true,
}
