package models

import "time"

// IssueState indicates whether an issue is open or closed.
type IssueState string

const (
	IssueOpen   IssueState = "OPEN"
	IssueClosed IssueState = "CLOSED"
)

// Label is a GitHub issue/PR label.
type Label struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description,omitempty"`
}

// Issue represents a GitHub issue.
type Issue struct {
	Number        int        `json:"number"`
	Title         string     `json:"title"`
	State         IssueState `json:"state"`
	Author        string     `json:"author"`
	Body          string     `json:"body"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	ClosedAt      *time.Time `json:"closed_at,omitempty"`
	Labels        []Label    `json:"labels,omitempty"`
	Assignees     []string   `json:"assignees,omitempty"`
	Comments      int        `json:"comments"`
	IsPullRequest bool       `json:"is_pull_request"`
}

// IssueComment represents a comment on an issue or PR.
type IssueComment struct {
	ID          int64     `json:"id"`
	Author      string    `json:"author"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	IsMinimized bool      `json:"is_minimized"`
}
