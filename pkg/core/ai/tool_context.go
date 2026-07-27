package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	gitpkg "github.com/zaidejjo/zgit/pkg/core/git"
)

// NewGetRepositoryContextTool returns a ToolDefinition that loads a read-only
// snapshot of the current repository state.
func NewGetRepositoryContextTool(gitClient gitpkg.GitAdapter) ToolDefinition {
	paramSchema := json.RawMessage(`{
		"type": "object",
		"properties": {},
		"description": "Load current repo context. No arguments needed."
	}`)

	return ToolDefinition{
		Name:        "get_repository_context",
		Description: "Load current branch, working tree status, recent commit log, unstaged diffs, and merge conflict files. Read-only, no side effects.",
		Parameters:  paramSchema,
		Handler: func(args json.RawMessage) *ToolResult {
			return getRepositoryContext(gitClient)
		},
	}
}

func getRepositoryContext(gitClient gitpkg.GitAdapter) *ToolResult {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rc := &RepositoryContext{}

	// Branch — handle unborn HEAD
	branch, err := gitClient.CurrentBranch(ctx)
	if err == nil {
		rc.Branch = branch
	}

	// Status
	status, err := gitClient.Status(ctx)
	if err == nil {
		rc.Status = status
	}

	// Recent log (last 20)
	log, err := gitClient.Log(ctx, gitpkg.LogOptions{Count: 20})
	if err == nil {
		rc.RecentLog = log
	}

	// Unstaged diffs
	diff, err := gitClient.Diff(ctx, gitpkg.DiffOptions{Unified: true})
	if err == nil {
		rc.UnstagedDiffs = diff.Files
	}

	// Conflict files
	conflicts, err := gitClient.ConflictFiles(ctx)
	if err == nil {
		rc.Conflicts = conflicts
	}

	data, err := json.Marshal(rc)
	if err != nil {
		return &ToolResult{Success: false, Error: fmt.Sprintf("marshal context: %v", err)}
	}

	return &ToolResult{Success: true, Data: data}
}
