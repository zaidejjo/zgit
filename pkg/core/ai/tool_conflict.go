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

// autoResolveConflictParams matches the tool's JSON Schema.
type autoResolveConflictParams struct {
	FilePath      string `json:"file_path"`
	BlockIndex    int    `json:"block_index"`    // -1 = all blocks
	PreferredSide string `json:"preferred_side"` // "ours" or "theirs"
}

// NewAutoResolveConflictTool creates a tool that resolves merge conflict blocks.
func NewAutoResolveConflictTool(gitClient gitpkg.GitAdapter) ToolDefinition {
	paramSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"file_path": {
				"type": "string",
				"description": "Path to the conflicted file"
			},
			"block_index": {
				"type": "integer",
				"description": "0-based conflict block index. -1 means resolve all blocks in this file.",
				"default": -1
			},
			"preferred_side": {
				"type": "string",
				"enum": ["ours", "theirs"],
				"description": "Which side to use for resolution"
			}
		},
		"required": ["file_path", "preferred_side"]
	}`)

	return ToolDefinition{
		Name:        "auto_resolve_conflict",
		Description: "Resolve a merge conflict block in a file by choosing ours or theirs. Requires user approval before execution.",
		Parameters:  paramSchema,
		Handler: func(args json.RawMessage) *ToolResult {
			var params autoResolveConflictParams
			if err := json.Unmarshal(args, &params); err != nil {
				return &ToolResult{Success: false, Error: fmt.Sprintf("invalid arguments: %v", err)}
			}
			return proposeConflictResolution(gitClient, params)
		},
	}
}

func proposeConflictResolution(gitClient gitpkg.GitAdapter, params autoResolveConflictParams) *ToolResult {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Fetch conflict detail to build preview
	detail, err := gitClient.GetMergeConflictDetail(ctx, params.FilePath)
	if err != nil {
		return &ToolResult{Success: false, Error: fmt.Sprintf("get conflict detail: %v", err)}
	}

	var preview strings.Builder
	preview.WriteString(fmt.Sprintf("Resolve %s conflicts in %s using '%s' side:\n",
		sideLabel(params.PreferredSide), params.FilePath, params.PreferredSide))

	if params.BlockIndex >= 0 && params.BlockIndex < len(detail.Blocks) {
		b := detail.Blocks[params.BlockIndex]
		preview.WriteString(fmt.Sprintf("  Block %d (lines %d-%d):\n", params.BlockIndex, b.OursStart, b.TheirsEnd))
		preview.WriteString(fmt.Sprintf("    OURS:   %s\n", truncateLines(b.Ours, 3)))
		preview.WriteString(fmt.Sprintf("    THEIRS: %s\n", truncateLines(b.Theirs, 3)))
	} else if params.BlockIndex == -1 {
		preview.WriteString(fmt.Sprintf("  All %d blocks will use '%s' side.\n", len(detail.Blocks), params.PreferredSide))
		for i, b := range detail.Blocks {
			preview.WriteString(fmt.Sprintf("  Block %d: ours=%d..%d  theirs=%d..%d\n",
				i, b.OursStart, b.OursEnd, b.TheirsStart, b.TheirsEnd))
		}
	} else {
		return &ToolResult{Success: false, Error: fmt.Sprintf("block index %d out of range (0-%d)", params.BlockIndex, len(detail.Blocks)-1)}
	}

	prop := &AgentActionProposal{
		Type:        "conflict_resolve",
		Description: fmt.Sprintf("Resolve %s in %s using '%s'", params.FilePath, params.FilePath, params.PreferredSide),
		Reasoning:   fmt.Sprintf("Apply '%s' side for %s conflict blocks in %s", params.PreferredSide, sideLabel(params.PreferredSide), params.FilePath),
		DiffPreview: preview.String(),
		Status:      ProposalPending,
		Params: map[string]any{
			"file_path":      params.FilePath,
			"block_index":    params.BlockIndex,
			"preferred_side": params.PreferredSide,
		},
	}

	data, _ := json.Marshal(map[string]string{"proposal_id": prop.ID})
	return &ToolResult{Success: true, Data: data, Proposal: prop}
}

func sideLabel(side string) string {
	if side == "ours" {
		return "our changes (HEAD)"
	}
	return "their changes (MERGE_HEAD)"
}

func truncateLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n") + fmt.Sprintf(" … (%d more lines)", len(lines)-n)
}

// ExecuteConflictResolution performs the actual resolution (called after approval).
func ExecuteConflictResolution(ctx context.Context, gitClient gitpkg.GitAdapter, params map[string]any) (string, error) {
	filePath, _ := params["file_path"].(string)
	preferredSide, _ := params["preferred_side"].(string)
	blockIndexFloat, _ := params["block_index"].(float64)
	blockIndex := int(blockIndexFloat)

	if filePath == "" || preferredSide == "" {
		return "", fmt.Errorf("invalid conflict resolution params")
	}

	if preferredSide == "ours" {
		if err := gitClient.CheckoutOurs(ctx, filePath); err != nil {
			return "", fmt.Errorf("checkout ours: %w", err)
		}
	} else if preferredSide == "theirs" {
		if err := gitClient.CheckoutTheirs(ctx, filePath); err != nil {
			return "", fmt.Errorf("checkout theirs: %w", err)
		}
	} else {
		return "", fmt.Errorf("invalid preferred_side: %s", preferredSide)
	}

	// Stage the resolved file
	detail, err := gitClient.GetMergeConflictDetail(ctx, filePath)
	if err != nil {
		return "", fmt.Errorf("get resolved content: %w", err)
	}
	if err := gitClient.StageResolvedFile(ctx, filePath, resolveFile(detail.RawContent, detail.Blocks, preferredSide, blockIndex)); err != nil {
		return "", fmt.Errorf("stage resolved file: %w", err)
	}

	label := sideLabel(preferredSide)
	return fmt.Sprintf("Resolved %s using %s", filePath, label), nil
}

// resolveFile reconstructs the file content by replacing conflict blocks with the chosen side.
func resolveFile(rawContent string, blocks []models.ConflictBlock, side string, blockIndex int) string {
	if blockIndex >= 0 && blockIndex < len(blocks) {
		// Resolve single block — just replace it in the raw content
		return replaceBlock(rawContent, blocks[blockIndex], side)
	}
	// Resolve all blocks
	result := rawContent
	for _, b := range blocks {
		result = replaceBlock(result, b, side)
	}
	return result
}

func replaceBlock(content string, block models.ConflictBlock, side string) string {
	chosen := block.Theirs
	if side == "ours" {
		chosen = block.Ours
	}

	// Build the conflict marker pattern to find and replace
	markerStart := "<<<<<<< " + "\n" + block.Ours + "\n=======\n" + block.Theirs + "\n>>>>>>> "
	idx := strings.Index(content, markerStart)
	if idx >= 0 {
		return content[:idx] + chosen + content[idx+len(markerStart):]
	}

	// Try without trailing space after markers
	markerStart2 := "<<<<<<<\n" + block.Ours + "\n=======\n" + block.Theirs + "\n>>>>>>>"
	idx2 := strings.Index(content, markerStart2)
	if idx2 >= 0 {
		return content[:idx2] + chosen + content[idx2+len(markerStart2):]
	}

	return content
}
