package models

// ConflictFile represents a single file with merge conflicts.
type ConflictFile struct {
	Path        string `json:"path"`
	AncestorSHA string `json:"ancestor_sha,omitempty"` // :1:file
	OursSHA     string `json:"ours_sha,omitempty"`     // :2:file
	TheirsSHA   string `json:"theirs_sha,omitempty"`   // :3:file
	BlockCount  int    `json:"block_count"`            // number of conflict blocks
}

// ConflictBlock represents a single conflict region in a file.
type ConflictBlock struct {
	Index       int    `json:"index"`              // block index (0-based)
	Ours        string `json:"ours"`               // lines between <<<<<<< and =======
	Theirs      string `json:"theirs"`             // lines between ======= and >>>>>>>
	OursStart   int    `json:"ours_start"`         // line number in our version
	OursEnd     int    `json:"ours_end"`           // end line in our version
	TheirsStart int    `json:"theirs_start"`       // line number in their version
	TheirsEnd   int    `json:"theirs_end"`         // end line in their version
	Resolved    string `json:"resolved,omitempty"` // user's chosen resolution
	State       string `json:"state"`              // "unresolved" | "use-ours" | "use-theirs" | "edited"
}

// MergeConflictDetail contains full conflict detail for a file.
type MergeConflictDetail struct {
	Path       string          `json:"path"`
	Ours       string          `json:"ours"`               // full file content from HEAD (stage 2)
	Theirs     string          `json:"theirs"`             // full file content from MERGE_HEAD (stage 3)
	Ancestor   string          `json:"ancestor,omitempty"` // full file content from merge-base (stage 1)
	RawContent string          `json:"raw_content"`        // working tree content WITH conflict markers
	Blocks     []ConflictBlock `json:"blocks"`             // parsed conflict blocks
	HasMerge   bool            `json:"has_merge"`          // has conflict markers
}

// StageResolvedFile request body (used internally).
type StageResolvedFileRequest struct {
	File            string `json:"file"`
	ResolvedContent string `json:"resolved_content"`
}
