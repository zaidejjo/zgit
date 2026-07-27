package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	gitpkg "github.com/zaidejjo/zgit/pkg/core/git"
	"github.com/zaidejjo/zgit/pkg/core/models"
)

// reviewParams matches the tool's JSON Schema.
type reviewParams struct {
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
}

// NewGeneratePRReviewTool creates a tool that loads the diff between two branches
// for AI-powered code review. Returns the diff as data so the LLM can produce
// a structured review in its response text.
func NewGeneratePRReviewTool(gitClient gitpkg.GitAdapter) ToolDefinition {
	paramSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"source_branch": {
				"type": "string",
				"description": "The source/feature branch containing the changes to review"
			},
			"target_branch": {
				"type": "string",
				"description": "The target/base branch to compare against (e.g. main, master)"
			}
		},
		"required": ["source_branch", "target_branch"]
	}`)

	return ToolDefinition{
		Name:        "generate_pr_review",
		Description: "Load diff between source and target branches for code review. Returns the diff content for analysis.",
		Parameters:  paramSchema,
		Handler: func(args json.RawMessage) *ToolResult {
			var params reviewParams
			if err := json.Unmarshal(args, &params); err != nil {
				return &ToolResult{Success: false, Error: fmt.Sprintf("invalid arguments: %v", err)}
			}
			return getBranchDiff(gitClient, params)
		},
	}
}

func getBranchDiff(gitClient gitpkg.GitAdapter, params reviewParams) *ToolResult {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get diff between target and source
	diff, err := gitClient.Diff(ctx, gitpkg.DiffOptions{
		A:       params.TargetBranch,
		B:       params.SourceBranch,
		Unified: true,
	})
	if err != nil {
		return &ToolResult{Success: false, Error: fmt.Sprintf("get diff: %v", err)}
	}

	if len(diff.Files) == 0 {
		return &ToolResult{Success: false, Error: fmt.Sprintf("no diff between %s and %s", params.SourceBranch, params.TargetBranch)}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Diff between %s (base) and %s (source):\n", params.TargetBranch, params.SourceBranch))
	b.WriteString(fmt.Sprintf("Total: %d files, +%d / -%d lines\n\n", len(diff.Files), diff.TotalAdds, diff.TotalDeletes))

	for _, f := range diff.Files {
		b.WriteString(fmt.Sprintf("File: %s (%s, +%d / -%d)\n",
			displayPath(f), changeTypeLabel(f.Type), f.Additions, f.Deletions))
		if f.UnifiedDiff != "" {
			b.WriteString(f.UnifiedDiff)
			b.WriteString("\n")
		}
	}

	diffText := b.String()
	const maxLen = 8000
	if len(diffText) > maxLen {
		diffText = diffText[:maxLen] + "\n... (diff truncated to " + fmt.Sprintf("%d chars)", maxLen)
	}

	data, err := json.Marshal(map[string]string{
		"source_branch": params.SourceBranch,
		"target_branch": params.TargetBranch,
		"diff":          diffText,
		"summary":       fmt.Sprintf("%d files changed, +%d / -%d", len(diff.Files), diff.TotalAdds, diff.TotalDeletes),
	})
	if err != nil {
		return &ToolResult{Success: false, Error: fmt.Sprintf("marshal review data: %v", err)}
	}

	return &ToolResult{Success: true, Data: data}
}

func displayPath(f models.FileChange) string {
	if f.OldPath != "" && f.OldPath != f.NewPath {
		return fmt.Sprintf("%s → %s", f.OldPath, f.NewPath)
	}
	return f.NewPath
}

func changeTypeLabel(t models.FileChangeType) string {
	switch t {
	case models.FileAdded:
		return "added"
	case models.FileModified:
		return "modified"
	case models.FileDeleted:
		return "deleted"
	case models.FileRenamed:
		return "renamed"
	case models.FileCopied:
		return "copied"
	default:
		return "changed"
	}
}
