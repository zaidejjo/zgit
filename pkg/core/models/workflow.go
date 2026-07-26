package models

import "time"

// Workflow represents a GitHub Actions workflow.
type Workflow struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`  // .github/workflows/ci.yml
	State     string    `json:"state"` // active, deleted, disabled
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WorkflowRunStatus indicates the current state of a run.
type WorkflowRunStatus string

const (
	RunQueued     WorkflowRunStatus = "QUEUED"
	RunInProgress WorkflowRunStatus = "IN_PROGRESS"
	RunCompleted  WorkflowRunStatus = "COMPLETED"
	RunCancelled  WorkflowRunStatus = "CANCELLED"
	RunSkipped    WorkflowRunStatus = "SKIPPED"
	RunStale      WorkflowRunStatus = "STALE"
	RunFailure    WorkflowRunStatus = "FAILURE"
	RunSuccess    WorkflowRunStatus = "SUCCESS"
)

// WorkflowRun represents a single workflow execution.
type WorkflowRun struct {
	ID           int64             `json:"id"`
	WorkflowName string            `json:"workflow_name"`
	Event        string            `json:"event"` // push, pull_request, workflow_dispatch
	Status       WorkflowRunStatus `json:"status"`
	Conclusion   string            `json:"conclusion,omitempty"`
	Branch       string            `json:"branch"`
	HeadSHA      string            `json:"head_sha"`
	RunNumber    int               `json:"run_number"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	HTMLURL      string            `json:"html_url"`
}

// Job represents a job within a workflow run.
type Job struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Conclusion  string    `json:"conclusion,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	RunnerName  string    `json:"runner_name,omitempty"`
	Steps       []Step    `json:"steps,omitempty"`
}

// Step represents a single step within a job.
type Step struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion,omitempty"`
	Number     int    `json:"number"`
}
