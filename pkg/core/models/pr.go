package models

import "time"

// PullRequestState is the current state of a pull request.
type PullRequestState string

const (
	PRStateOpen   PullRequestState = "OPEN"
	PRStateClosed PullRequestState = "CLOSED"
	PRStateMerged PullRequestState = "MERGED"
	PRStateDraft  PullRequestState = "DRAFT"
)

// PullRequestSummary is a lightweight PR representation for lists.
type PullRequestSummary struct {
	Number      int              `json:"number"`
	Title       string           `json:"title"`
	State       PullRequestState `json:"state"`
	Author      string           `json:"author"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	IsDraft     bool             `json:"is_draft"`
	Mergeable   string           `json:"mergeable"` // MERGEABLE, CONFLICTING, UNKNOWN
	HeadRef     string           `json:"head_ref"`
	BaseRef     string           `json:"base_ref"`
	StatusEmoji string           `json:"status_emoji"` // ✓ / ✗ / pending
	ReviewState string           `json:"review_state"` // APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED
	Labels      []string         `json:"labels,omitempty"`
}

// PullRequestDetail is a full PR with nested data.
type PullRequestDetail struct {
	PullRequestSummary
	Body         string       `json:"body"`
	ClosedAt     *time.Time   `json:"closed_at,omitempty"`
	MergedAt     *time.Time   `json:"merged_at,omitempty"`
	MergedBy     string       `json:"merged_by,omitempty"`
	Additions    int          `json:"additions"`
	Deletions    int          `json:"deletions"`
	ChangedFiles int          `json:"changed_files"`
	Commits      []Commit     `json:"commits,omitempty"`
	Reviews      []Review     `json:"reviews,omitempty"`
	CheckRuns    []CheckRun   `json:"check_runs,omitempty"`
	Files        []FileChange `json:"files,omitempty"`
	Comments     int          `json:"comments"`
}

// ReviewState indicates the outcome of a PR review.
type ReviewState string

const (
	ReviewApproved         ReviewState = "APPROVED"
	ReviewChangesRequested ReviewState = "CHANGES_REQUESTED"
	ReviewCommented        ReviewState = "COMMENTED"
	ReviewPending          ReviewState = "PENDING"
	ReviewDismissed        ReviewState = "DISMISSED"
)

// Review represents a GitHub PR review.
type Review struct {
	ID        int64       `json:"id"`
	Author    string      `json:"author"`
	State     ReviewState `json:"state"`
	Body      string      `json:"body"`
	Submitted time.Time   `json:"submitted_at"`
}

// CheckRun represents a status check or check run on a PR.
type CheckRun struct {
	Name       string `json:"name"`
	State      string `json:"state"`      // QUEUED, IN_PROGRESS, COMPLETED
	Conclusion string `json:"conclusion"` // SUCCESS, FAILURE, NEUTRAL, CANCELLED, SKIPPED
	DetailsURL string `json:"details_url,omitempty"`
}
