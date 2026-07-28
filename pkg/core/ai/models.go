package ai

import (
	"encoding/json"
	"time"

	"github.com/zaidejjo/zgit/pkg/core/models"
)

// Message represents a chat message in the agent conversation.
type Message struct {
	Role       string     `json:"role"`                   // "system", "user", "assistant", "tool"
	Content    string     `json:"content,omitempty"`      // text content
	ToolCallID string     `json:"tool_call_id,omitempty"` // for tool result messages
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // for assistant messages
}

// ToolCall represents a function call requested by the LLM.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string of args
}

// ToolDefinition describes a callable tool the agent may invoke.
// The Handler field is the Go function; it is not serialized.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"` // JSON Schema
	Handler     ToolHandler     `json:"-"`          // Go implementation
}

// ToolHandler executes a tool with parsed arguments and returns a result.
type ToolHandler func(args json.RawMessage) *ToolResult

// ToolResult is the outcome of a single tool invocation.
type ToolResult struct {
	Success  bool                 `json:"success"`
	Data     json.RawMessage      `json:"data,omitempty"`
	Error    string               `json:"error,omitempty"`
	Proposal *AgentActionProposal `json:"proposal,omitempty"` // set when tool creates a proposal
}

// AgentResponse is returned by Agent.Chat after processing user input.
type AgentResponse struct {
	Message   string                `json:"message"`
	Proposals []AgentActionProposal `json:"proposals,omitempty"`
	Finished  bool                  `json:"finished"` // agent has completed its plan
}

// AgentActionProposal represents a git action awaiting user confirmation.
type AgentActionProposal struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`        // action type enum
	Description string         `json:"description"` // human-readable summary
	Reasoning   string         `json:"reasoning"`   // why this action is needed
	DiffPreview string         `json:"diff_preview,omitempty"`
	Status      ProposalStatus `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	Params      map[string]any `json:"params,omitempty"` // action-specific params
}

// ProposalStatus tracks the lifecycle of a proposed action.
type ProposalStatus string

const (
	ProposalPending  ProposalStatus = "pending"
	ProposalApproved ProposalStatus = "approved"
	ProposalRejected ProposalStatus = "rejected"
	ProposalExecuted ProposalStatus = "executed"
	ProposalFailed   ProposalStatus = "failed"
)

// RepositoryContext is a read-only snapshot of the current repository state.
type RepositoryContext struct {
	Branch        string                `json:"branch"`
	Status        *models.Status        `json:"status,omitempty"`
	RecentLog     []*models.Commit      `json:"recent_log,omitempty"`
	UnstagedDiffs []models.FileChange   `json:"unstaged_diffs,omitempty"`
	Conflicts     []models.ConflictFile `json:"conflicts,omitempty"`
}

// ProposalResult is returned after executing or rejecting a proposal.
type ProposalResult struct {
	ProposalID string         `json:"proposal_id"`
	Status     ProposalStatus `json:"status"`
	Success    bool           `json:"success"`
	Output     string         `json:"output,omitempty"`
	Error      string         `json:"error,omitempty"`
}

// AgentConfig wraps AI config with agent-specific settings.
type AgentConfig struct {
	Provider ProviderKind `json:"provider"`
	APIKey   string       `json:"api_key"`
	Model    string       `json:"model"`
	Endpoint string       `json:"endpoint,omitempty"`
	MaxTurns int          `json:"max_turns"` // max conversation iterations (default 10)
	AutoMode bool         `json:"auto_mode"` // skip proposals for safe write actions
}

// AsConfig converts AgentConfig to the base ai.Config for provider init.
func (ac AgentConfig) AsConfig() Config {
	return Config{
		Provider: ac.Provider,
		APIKey:   ac.APIKey,
		Model:    ac.Model,
		Endpoint: ac.Endpoint,
	}
}
