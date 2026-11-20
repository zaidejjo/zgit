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

// CommitContext holds repository state used to inform AI commit message generation.
// Including branch name and recent commit history lets the AI match project conventions.
type CommitContext struct {
	Branch         string   `json:"branch"`          // current branch name
	RecentMessages []string `json:"recent_messages"` // last N commit messages (style matching)
	StatusSummary  string   `json:"status_summary"`  // e.g. "3 staged, 2 unstaged, 1 untracked"
	ChangedFiles   int      `json:"changed_files"`   // number of files changed in diff
	RepoPath       string   `json:"repo_path"`       // absolute repo path
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

// ModelInfo describes a model's capabilities and pricing for UI badges.
type ModelInfo struct {
	ID             string `json:"id"`              // model identifier (e.g. "claude-sonnet-4-20250514")
	ContextWindow  int    `json:"context_window"`  // max context tokens (0 = unknown)
	SupportsTools  bool   `json:"supports_tools"`  // supports function/tool calling
	SupportsVision bool   `json:"supports_vision"` // supports image input
	PricingInput   string `json:"pricing_input"`   // per 1M input tokens (e.g. "$3.00")
	PricingOutput  string `json:"pricing_output"`  // per 1M output tokens
	BestFor        string `json:"best_for"`        // ideal use case (e.g. "commit messages", "code review")
}

// DefaultModelInfo is a hardcoded registry of well-known models.
// Used as the primary source; models.dev is a supplemental fallback.
var DefaultModelInfo = map[string]ModelInfo{
	// OpenAI
	"gpt-4o-mini":   {ID: "gpt-4o-mini", ContextWindow: 128000, SupportsTools: true, SupportsVision: true, PricingInput: "$0.15", PricingOutput: "$0.60", BestFor: "commit messages, general"},
	"gpt-4o":        {ID: "gpt-4o", ContextWindow: 128000, SupportsTools: true, SupportsVision: true, PricingInput: "$2.50", PricingOutput: "$10.00", BestFor: "code review, complex reasoning"},
	"gpt-4-turbo":   {ID: "gpt-4-turbo", ContextWindow: 128000, SupportsTools: true, SupportsVision: true, PricingInput: "$10.00", PricingOutput: "$30.00", BestFor: "complex reasoning"},
	"gpt-3.5-turbo": {ID: "gpt-3.5-turbo", ContextWindow: 16384, SupportsTools: true, SupportsVision: false, PricingInput: "$0.50", PricingOutput: "$1.50", BestFor: "simple tasks"},

	// Anthropic
	"claude-sonnet-4-20250514":   {ID: "claude-sonnet-4-20250514", ContextWindow: 200000, SupportsTools: true, SupportsVision: true, PricingInput: "$3.00", PricingOutput: "$15.00", BestFor: "code review, complex reasoning"},
	"claude-3-5-sonnet-20241022": {ID: "claude-3-5-sonnet-20241022", ContextWindow: 200000, SupportsTools: true, SupportsVision: true, PricingInput: "$3.00", PricingOutput: "$15.00", BestFor: "code review, complex reasoning"},
	"claude-3-opus-20240229":     {ID: "claude-3-opus-20240229", ContextWindow: 200000, SupportsTools: true, SupportsVision: true, PricingInput: "$15.00", PricingOutput: "$75.00", BestFor: "complex reasoning"},
	"claude-3-haiku-20240307":    {ID: "claude-3-haiku-20240307", ContextWindow: 200000, SupportsTools: true, SupportsVision: true, PricingInput: "$0.25", PricingOutput: "$1.25", BestFor: "commit messages, fast tasks"},

	// DeepSeek
	"deepseek-chat":     {ID: "deepseek-chat", ContextWindow: 128000, SupportsTools: true, SupportsVision: false, PricingInput: "$0.27", PricingOutput: "$1.10", BestFor: "code tasks"},
	"deepseek-coder":    {ID: "deepseek-coder", ContextWindow: 16384, SupportsTools: true, SupportsVision: false, PricingInput: "$0.14", PricingOutput: "$0.28", BestFor: "code generation"},
	"deepseek-reasoner": {ID: "deepseek-reasoner", ContextWindow: 64000, SupportsTools: true, SupportsVision: false, PricingInput: "$0.55", PricingOutput: "$2.19", BestFor: "complex reasoning"},

	// OpenRouter (openai/gpt-4o-mini style)
	"openai/gpt-4o-mini":          {ID: "openai/gpt-4o-mini", ContextWindow: 128000, SupportsTools: true, SupportsVision: true, PricingInput: "$0.15", PricingOutput: "$0.60", BestFor: "commit messages, general"},
	"anthropic/claude-3.5-sonnet": {ID: "anthropic/claude-3.5-sonnet", ContextWindow: 200000, SupportsTools: true, SupportsVision: true, PricingInput: "$3.00", PricingOutput: "$15.00", BestFor: "code review"},
	"deepseek/deepseek-v4-flash":  {ID: "deepseek/deepseek-v4-flash", ContextWindow: 128000, SupportsTools: true, SupportsVision: false, PricingInput: "$0.27", PricingOutput: "$1.10", BestFor: "code tasks"},

	// Ollama (local)
	"llama3.2": {ID: "llama3.2", ContextWindow: 32768, SupportsTools: true, SupportsVision: false, PricingInput: "$0.00", PricingOutput: "$0.00", BestFor: "local, private"},
	"llama3.1": {ID: "llama3.1", ContextWindow: 128000, SupportsTools: true, SupportsVision: false, PricingInput: "$0.00", PricingOutput: "$0.00", BestFor: "local, private"},
}

// GetModelInfo returns ModelInfo for a model ID, falling back to a minimal entry.
func GetModelInfo(modelID string) ModelInfo {
	if info, ok := DefaultModelInfo[modelID]; ok {
		return info
	}
	return ModelInfo{ID: modelID}
}
