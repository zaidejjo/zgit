package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	gitpkg "github.com/zaidejjo/zgit/pkg/core/git"
)

// Agent orchestrates multi-turn AI conversations with tool calling.
// All repo-modifying actions flow through AgentActionProposal for user confirmation.
type Agent struct {
	mu       sync.Mutex
	cfg      Config
	git      gitpkg.GitAdapter
	provider ToolCallingProvider

	messages []Message
	tools    []ToolDefinition
	toolMap  map[string]ToolDefinition

	proposals  []AgentActionProposal
	nextPropID int

	systemPrompt string
}

// NewAgent creates an agent wired to the given git adapter and provider config.
func NewAgent(cfg Config, gitClient gitpkg.GitAdapter) (*Agent, error) {
	provider, err := NewToolCallingProvider(cfg)
	if err != nil {
		return nil, err
	}
	maxTurns := cfg.MaxTurns
	if maxTurns <= 0 {
		maxTurns = DefaultMaxTurns
	}

	a := &Agent{
		cfg:      cfg,
		git:      gitClient,
		provider: provider,
		tools:    make([]ToolDefinition, 0),
		toolMap:  make(map[string]ToolDefinition),
		messages: make([]Message, 0),
	}

	// Register built-in tools
	a.registerTool(NewGetRepositoryContextTool(gitClient))
	a.RegisterTool(NewSuggestGitCommandTool(gitClient)) // uses a.git for approval execution

	// Conflict and review tools are registered via RegisterTool after construction
	// so the caller can decide which to include.

	a.systemPrompt = buildAgentSystemPrompt()
	a.messages = append(a.messages, Message{Role: "system", Content: a.systemPrompt})

	_ = maxTurns // used in Chat loop
	return a, nil
}

func (a *Agent) registerTool(td ToolDefinition) {
	a.tools = append(a.tools, td)
	a.toolMap[td.Name] = td
}

// RegisterTool adds a tool to the agent's available tools.
func (a *Agent) RegisterTool(td ToolDefinition) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.registerTool(td)
}

// GetTools returns the current set of registered tools.
func (a *Agent) GetTools() []ToolDefinition {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]ToolDefinition, len(a.tools))
	copy(out, a.tools)
	return out
}

// Chat sends a user message and returns the agent's response.
func (a *Agent) Chat(ctx context.Context, userMessage string) (*AgentResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.messages = append(a.messages, Message{Role: "user", Content: userMessage})

	maxTurns := a.cfg.MaxTurns
	if maxTurns <= 0 {
		maxTurns = DefaultMaxTurns
	}

	hasPendingProposal := false

	for turn := 0; turn < maxTurns; turn++ {
		llmMsg, err := a.provider.Chat(ctx, a.messages, a.tools)
		if err != nil {
			return nil, fmt.Errorf("LLM call failed (turn %d): %w", turn, err)
		}
		a.messages = append(a.messages, llmMsg)

		// LLM responded with text and no tool calls — done if no pending proposals
		if len(llmMsg.ToolCalls) == 0 {
			resp := &AgentResponse{
				Message:  llmMsg.Content,
				Finished: !hasPendingProposal,
			}
			if hasPendingProposal {
				resp.Proposals = a.pendingProposals()
			}
			return resp, nil
		}

		// Process tool calls
		for _, tc := range llmMsg.ToolCalls {
			toolDef, ok := a.toolMap[tc.Name]
			if !ok {
				errMsg := fmt.Sprintf("unknown tool: %s", tc.Name)
				a.messages = append(a.messages, Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    fmt.Sprintf(`{"success":false,"error":%q}`, errMsg),
				})
				continue
			}

			var args json.RawMessage
			if tc.Arguments != "" {
				args = json.RawMessage(tc.Arguments)
			} else {
				args = json.RawMessage(`{}`)
			}

			result := toolDef.Handler(args)

			// If tool created a proposal, store it and mark pause
			if result.Proposal != nil {
				result.Proposal.ID = a.nextProposalID()
				result.Proposal.CreatedAt = time.Now()
				a.proposals = append(a.proposals, *result.Proposal)
				hasPendingProposal = true

				// Return proposal info as tool result
				propData, _ := json.Marshal(map[string]string{
					"proposal_id": result.Proposal.ID,
					"status":      string(result.Proposal.Status),
					"description": result.Proposal.Description,
				})
				a.messages = append(a.messages, Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    string(propData),
				})
				continue
			}

			// Normal tool result
			resultData := "{}"
			if result.Data != nil {
				resultData = string(result.Data)
			}
			if !result.Success {
				resultData = fmt.Sprintf(`{"success":false,"error":%q}`, result.Error)
			}
			a.messages = append(a.messages, Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    resultData,
			})
		}

		// If we got proposals this turn, pause and return
		if hasPendingProposal {
			return &AgentResponse{
				Message:   "I've prepared some actions that need your approval.",
				Proposals: a.pendingProposals(),
				Finished:  false,
			}, nil
		}
	}

	// Max turns exhausted — agent should release control
	return &AgentResponse{
		Message:  "I've reached the maximum number of turns for this request. Please approve or reject the pending proposals, or start a new conversation.",
		Finished: true,
	}, nil
}

// pendingProposals returns proposals that are still pending.
func (a *Agent) pendingProposals() []AgentActionProposal {
	var out []AgentActionProposal
	for _, p := range a.proposals {
		if p.Status == ProposalPending {
			out = append(out, p)
		}
	}
	return out
}

func (a *Agent) nextProposalID() string {
	a.nextPropID++
	return fmt.Sprintf("prop_%d", a.nextPropID)
}

// ApproveProposal executes an approved proposal and returns the result.
func (a *Agent) ApproveProposal(ctx context.Context, proposalID string) (*ProposalResult, error) {
	a.mu.Lock()

	// Find the proposal
	idx := -1
	for i, p := range a.proposals {
		if p.ID == proposalID && p.Status == ProposalPending {
			idx = i
			break
		}
	}
	if idx == -1 {
		a.mu.Unlock()
		return nil, fmt.Errorf("proposal %q not found or not pending", proposalID)
	}

	proposal := &a.proposals[idx]
	proposal.Status = ProposalApproved
	a.mu.Unlock()

	// Execute based on action type
	output, err := a.executeAction(ctx, proposal)

	a.mu.Lock()
	defer a.mu.Unlock()

	// Update proposal status
	for i := range a.proposals {
		if a.proposals[i].ID == proposalID {
			if err != nil {
				a.proposals[i].Status = ProposalFailed
			} else {
				a.proposals[i].Status = ProposalExecuted
			}
			break
		}
	}

	pr := &ProposalResult{
		ProposalID: proposalID,
		Success:    err == nil,
		Output:     output,
	}
	if err != nil {
		pr.Status = ProposalFailed
		pr.Error = err.Error()
	} else {
		pr.Status = ProposalExecuted
	}

	// Add result to conversation
	if err != nil {
		a.messages = append(a.messages, Message{
			Role:    "user",
			Content: fmt.Sprintf("[Proposal %s executed] Result: %s", proposalID, output),
		})
	} else {
		a.messages = append(a.messages, Message{
			Role:    "user",
			Content: fmt.Sprintf("[System: Proposal %s failed] %s", proposalID, err.Error()),
		})
	}

	return pr, nil
}

// RejectProposal marks a proposal as rejected, with optional feedback for the agent.
func (a *Agent) RejectProposal(ctx context.Context, proposalID string, feedback string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	found := false
	for i := range a.proposals {
		if a.proposals[i].ID == proposalID && a.proposals[i].Status == ProposalPending {
			a.proposals[i].Status = ProposalRejected
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("proposal %q not found or not pending", proposalID)
	}

	msg := fmt.Sprintf("[Proposal %s rejected]", proposalID)
	if feedback != "" {
		msg += " Feedback: " + feedback
	}
	a.messages = append(a.messages, Message{Role: "user", Content: msg})
	return nil
}

// Reset clears conversation history and all proposals.
func (a *Agent) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.messages = make([]Message, 0)
	a.messages = append(a.messages, Message{Role: "system", Content: a.systemPrompt})
	a.proposals = make([]AgentActionProposal, 0)
	a.nextPropID = 0
}

// History returns the raw conversation messages (for frontend display).
func (a *Agent) History() []Message {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]Message, len(a.messages))
	copy(out, a.messages)
	return out
}

// GetPendingProposals returns all pending proposals.
func (a *Agent) GetPendingProposals() []AgentActionProposal {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.pendingProposals()
}

// executeAction maps a proposal's action_type + params to the correct GitAdapter method.
func (a *Agent) executeAction(ctx context.Context, proposal *AgentActionProposal) (string, error) {
	actionType := proposal.Type
	params := proposal.Params

	switch actionType {
	case "create_branch":
		name, _ := params["name"].(string)
		if name == "" {
			return "", fmt.Errorf("create_branch requires 'name' param")
		}
		if err := a.git.BranchCreate(ctx, name); err != nil {
			return "", err
		}
		return fmt.Sprintf("Created branch %s", name), nil

	case "checkout":
		ref, _ := params["ref"].(string)
		if ref == "" {
			return "", fmt.Errorf("checkout requires 'ref' param")
		}
		if err := a.git.Checkout(ctx, ref); err != nil {
			return "", err
		}
		return fmt.Sprintf("Checked out %s", ref), nil

	case "create_branch_and_checkout":
		name, _ := params["name"].(string)
		if name == "" {
			return "", fmt.Errorf("create_branch_and_checkout requires 'name' param")
		}
		if err := a.git.CreateBranchAndCheckout(ctx, name); err != nil {
			return "", err
		}
		return fmt.Sprintf("Created and switched to branch %s", name), nil

	case "commit":
		message, _ := params["message"].(string)
		if message == "" {
			return "", fmt.Errorf("commit requires 'message' param")
		}
		body, _ := params["body"].(string)
		opts := gitpkg.CommitOptions{Message: message, Body: body}
		if err := a.git.Commit(ctx, opts); err != nil {
			return "", err
		}
		return fmt.Sprintf("Committed: %s", message), nil

	case "push":
		remote, _ := params["remote"].(string)
		branch, _ := params["branch"].(string)
		force, _ := params["force"].(bool)
		opts := gitpkg.PushOptions{Remote: remote, Branch: branch, Force: force}
		if remote == "" {
			opts.Remote = "origin"
		}
		if err := a.git.Push(ctx, opts); err != nil {
			return "", err
		}
		return "Push completed", nil

	case "pull":
		rebase, _ := params["rebase"].(bool)
		opts := gitpkg.PullOptions{Rebase: rebase}
		if err := a.git.Pull(ctx, opts); err != nil {
			return "", err
		}
		return "Pull completed", nil

	case "stash_push":
		msg, _ := params["message"].(string)
		includeUntracked, _ := params["include_untracked"].(bool)
		opts := gitpkg.StashOptions{IncludeUntracked: includeUntracked}
		if err := a.git.StashPush(ctx, opts, msg); err != nil {
			return "", err
		}
		return "Stashed changes", nil

	case "stash_pop":
		index, _ := params["index"].(float64)
		if err := a.git.StashPop(ctx, int(index)); err != nil {
			return "", err
		}
		return fmt.Sprintf("Popped stash @%d", int(index)), nil

	case "reset_commit":
		sha, _ := params["sha"].(string)
		mode, _ := params["mode"].(string)
		if mode == "" {
			mode = "mixed"
		}
		if err := a.git.ResetCommit(ctx, sha, mode); err != nil {
			return "", err
		}
		return fmt.Sprintf("Reset to %s (%s)", sha, mode), nil

	case "revert_commit":
		sha, _ := params["sha"].(string)
		if sha == "" {
			return "", fmt.Errorf("revert_commit requires 'sha' param")
		}
		if err := a.git.Revert(ctx, sha); err != nil {
			return "", err
		}
		return fmt.Sprintf("Reverted %s", sha), nil

	case "merge":
		branch, _ := params["branch"].(string)
		if branch == "" {
			return "", fmt.Errorf("merge requires 'branch' param")
		}
		if _, err := a.git.Merge(ctx, branch); err != nil {
			return "", err
		}
		return fmt.Sprintf("Merged %s", branch), nil

	case "delete_branch":
		name, _ := params["name"].(string)
		force, _ := params["force"].(bool)
		if name == "" {
			return "", fmt.Errorf("delete_branch requires 'name' param")
		}
		if err := a.git.BranchDelete(ctx, name, force); err != nil {
			return "", err
		}
		return fmt.Sprintf("Deleted branch %s", name), nil

	case "stage_all":
		opts := gitpkg.AddOptions{All: true}
		if err := a.git.Add(ctx, opts); err != nil {
			return "", err
		}
		return "Staged all changes", nil

	case "unstage_all":
		if err := a.git.Reset(ctx); err != nil {
			return "", err
		}
		return "Unstaged all changes", nil

	case "discard_file":
		file, _ := params["file"].(string)
		if file == "" {
			return "", fmt.Errorf("discard_file requires 'file' param")
		}
		if err := a.git.Restore(ctx, file); err != nil {
			return "", err
		}
		return fmt.Sprintf("Discarded changes in %s", file), nil

	case "discard_all":
		// Get status first to know which files to restore
		status, err := a.git.Status(ctx)
		if err != nil {
			return "", fmt.Errorf("get status: %w", err)
		}
		for _, f := range status.UnstagedFiles() {
			if err := a.git.Restore(ctx, f.Path); err != nil {
				return "", err
			}
		}
		return "Discarded all unstaged changes", nil

	case "conflict_resolve":
		return ExecuteConflictResolution(ctx, a.git, params)

	case "tag_create":
		name, _ := params["name"].(string)
		target, _ := params["target"].(string)
		message, _ := params["message"].(string)
		if name == "" {
			return "", fmt.Errorf("tag_create requires 'name' param")
		}
		if err := a.git.TagCreate(ctx, name, target, message); err != nil {
			return "", err
		}
		return fmt.Sprintf("Created tag %s", name), nil

	case "tag_delete":
		name, _ := params["name"].(string)
		if name == "" {
			return "", fmt.Errorf("tag_delete requires 'name' param")
		}
		if err := a.git.TagDelete(ctx, name); err != nil {
			return "", err
		}
		return fmt.Sprintf("Deleted tag %s", name), nil

	default:
		return "", fmt.Errorf("unknown action type: %s", actionType)
	}
}

// --- SuggestGitCommand tool ---

// suggestGitCommandParams matches the tool's JSON Schema.
type suggestGitCommandParams struct {
	ActionType  string         `json:"action_type"`
	Description string         `json:"description"`
	Reasoning   string         `json:"reasoning"`
	Params      map[string]any `json:"params,omitempty"`
}

// NewSuggestGitCommandTool creates the tool that proposes safe git actions.
func NewSuggestGitCommandTool(gitClient gitpkg.GitAdapter) ToolDefinition {
	paramSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"action_type": {
				"type": "string",
				"enum": ["create_branch", "checkout", "create_branch_and_checkout", "commit", "push", "pull", "stash_push", "stash_pop", "reset_commit", "revert_commit", "merge", "delete_branch", "stage_all", "unstage_all", "discard_file", "discard_all", "tag_create", "tag_delete", "conflict_resolve"],
				"description": "Type of git action to perform"
			},
			"description": {
				"type": "string",
				"description": "Human-readable summary of what the action does"
			},
			"reasoning": {
				"type": "string",
				"description": "Why this action is needed based on the context"
			},
			"params": {
				"type": "object",
				"description": "Action-specific parameters (branch name, commit message, etc.)",
				"additionalProperties": true
			}
		},
		"required": ["action_type", "description", "reasoning"]
	}`)

	return ToolDefinition{
		Name:        "suggest_git_command",
		Description: "Propose a safe git operation to perform. The agent will NOT execute until you approve. Use this for any action that modifies repository state (branching, committing, pushing, stashing, resolving conflicts, etc.).",
		Parameters:  paramSchema,
		Handler: func(args json.RawMessage) *ToolResult {
			var params suggestGitCommandParams
			if err := json.Unmarshal(args, &params); err != nil {
				return &ToolResult{Success: false, Error: fmt.Sprintf("invalid arguments: %v", err)}
			}
			if params.ActionType == "" || params.Description == "" || params.Reasoning == "" {
				return &ToolResult{Success: false, Error: "action_type, description, and reasoning are required"}
			}
			return &ToolResult{
				Success: true,
				Proposal: &AgentActionProposal{
					Type:        params.ActionType,
					Description: params.Description,
					Reasoning:   params.Reasoning,
					Status:      ProposalPending,
					Params:      params.Params,
				},
			}
		},
	}
}

// buildAgentSystemPrompt returns the system prompt for the agentic AI assistant.
func buildAgentSystemPrompt() string {
	return `You are an autonomous Git AI assistant. Your purpose is to help users manage their Git repositories safely and efficiently.

AVAILABLE TOOLS:
1. get_repository_context — Load current repo state (branch, status, log, diffs, conflicts). Read-only. Call this first to understand the repo.
2. suggest_git_command — Propose a git operation. This creates a proposal the user must approve before execution. Use for ALL write operations.
3. auto_resolve_conflict — Propose conflict resolution for a merge conflict file. Creates a proposal requiring user approval.
4. generate_pr_review — Load diff between two branches for code review. Read-only.

RULES:
- ALWAYS call get_repository_context first to understand the user's current state.
- For ANY action that modifies the repository (branching, committing, pushing, merging, stashing, resolving conflicts, etc.), use suggest_git_command. DO NOT suggest manual git commands — always use the structured action tool.
- Provide clear reasoning for each proposed action explaining WHY it's needed.
- If you need to check the result of a command before suggesting the next step, explain that the user should approve and then you'll check the state.
- Be concise. Focus on what needs to happen and why.
- After a proposal is approved and executed, call get_repository_context again to verify the result and continue the plan if needed.

ACTION TYPES and their required params:
- create_branch: {"name": "branch-name"}
- checkout: {"ref": "branch-name"}
- create_branch_and_checkout: {"name": "branch-name"}
- commit: {"message": "commit message", "body": "optional body"}
- push: {"remote": "origin", "branch": "main", "force": false}
- pull: {"rebase": true}
- stash_push: {"message": "optional", "include_untracked": false}
- stash_pop: {"index": 0}
- reset_commit: {"sha": "abc123", "mode": "mixed"}
- revert_commit: {"sha": "abc123"}
- merge: {"branch": "feature/xyz"}
- delete_branch: {"name": "old-branch", "force": false}
- stage_all: (no params needed)
- unstage_all: (no params needed)
- discard_file: {"file": "path/to/file.go"}
- discard_all: (no params needed)
- tag_create: {"name": "v1.0", "target": "main", "message": "optional"}
- tag_delete: {"name": "v1.0"}
- conflict_resolve: {"file_path": "path/to/file.go", "block_index": -1, "preferred_side": "ours"}

CRITICAL SAFETY RULES:
- NEVER execute destructive actions without user approval (always use suggest_git_command).
- NEVER propose force push unless the user explicitly asks.
- NEVER propose reset --hard unless the user explicitly asks.
- Always explain what the command will do before proposing it.`
}
