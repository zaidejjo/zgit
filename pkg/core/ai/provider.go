// Package ai provides multi-provider commit message generation.
package ai

import (
	"context"
	"fmt"
)

// ProviderKind enumerates supported AI providers.
type ProviderKind string

const (
	ProviderOpenAI      ProviderKind = "openai"
	ProviderAnthropic   ProviderKind = "anthropic"
	ProviderDeepSeek    ProviderKind = "deepseek"
	ProviderOpenRouter  ProviderKind = "openrouter"
	ProviderGroq        ProviderKind = "groq"
	ProviderOllama      ProviderKind = "ollama"
	ProviderOpenCodeZen ProviderKind = "opencodezen"
	ProviderCustom      ProviderKind = "custom"
)

// Config holds AI provider settings.
type Config struct {
	Provider ProviderKind `json:"provider"`
	APIKey   string       `json:"api_key"`
	Model    string       `json:"model"`
	Endpoint string       `json:"endpoint,omitempty"`  // custom endpoint URL
	MaxTurns int          `json:"max_turns,omitempty"` // agent max iterations (default 10)
	AutoMode bool         `json:"auto_mode,omitempty"` // skip proposals for safe actions
}

const DefaultMaxTurns = 10

// DefaultModels maps providers to their default model names.
var DefaultModels = map[ProviderKind]string{
	ProviderOpenAI:      "gpt-4o-mini",
	ProviderAnthropic:   "claude-sonnet-4-20250514",
	ProviderDeepSeek:    "deepseek/deepseek-v4-flash",
	ProviderOpenRouter:  "openai/gpt-4o-mini",
	ProviderGroq:        "llama-3.1-8b-instant",
	ProviderOllama:      "llama3.2",
	ProviderOpenCodeZen: "",
	ProviderCustom:      "",
}

// DefaultEndpoints maps providers to their API endpoints.
var DefaultEndpoints = map[ProviderKind]string{
	ProviderOpenAI:      "https://api.openai.com/v1/chat/completions",
	ProviderAnthropic:   "https://api.anthropic.com/v1/messages",
	ProviderDeepSeek:    "https://api.deepseek.com/chat/completions",
	ProviderOpenRouter:  "https://openrouter.ai/api/v1/chat/completions",
	ProviderGroq:        "https://api.groq.com/openai/v1/chat/completions",
	ProviderOllama:      "http://localhost:11434/v1/chat/completions",
	ProviderOpenCodeZen: "",
	ProviderCustom:      "",
}

// Generator generates commit messages from diffs.
type Generator interface {
	// GenerateCommitMessage sends the diff to the AI provider and returns a conventional commit message.
	GenerateCommitMessage(ctx context.Context, diff string, cfg Config) (string, error)
}

// NewGenerator creates the appropriate provider based on config.
func NewGenerator(cfg Config) (Generator, error) {
	switch cfg.Provider {
	case ProviderOpenAI, ProviderDeepSeek, ProviderOpenRouter, ProviderGroq, ProviderOllama, ProviderOpenCodeZen, ProviderCustom:
		return NewOpenAI(cfg), nil
	case ProviderAnthropic:
		return NewAnthropic(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", cfg.Provider)
	}
}

// AskProvider supports simple Q&A without tool calling.
// Used for "Ask" mode — read-only context queries.
type AskProvider interface {
	// Ask sends messages without tools and returns the assistant response.
	Ask(ctx context.Context, messages []Message) (Message, error)

	// AskStream streams tokens via onToken callback and returns the full response.
	AskStream(ctx context.Context, messages []Message, onToken func(string)) (Message, error)
}

// NewAskProvider creates the appropriate AskProvider based on config.
func NewAskProvider(cfg Config) (AskProvider, error) {
	switch cfg.Provider {
	case ProviderOpenAI, ProviderDeepSeek, ProviderOpenRouter, ProviderGroq, ProviderOllama, ProviderOpenCodeZen, ProviderCustom:
		return NewOpenAI(cfg), nil
	case ProviderAnthropic:
		return NewAnthropic(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", cfg.Provider)
	}
}

// buildSystemPrompt returns the system prompt for generating conventional commits.
func buildSystemPrompt() string {
	return `You are a git commit message generator. Generate a concise, accurate commit message following the Conventional Commits format:

<type>(<scope>): <description>

<optional body>

Types: feat, fix, docs, style, refactor, test, chore, perf, ci, build, revert
Rules:
- Subject line max 72 characters
- Use imperative mood ("Add feature" not "Added feature")
- Do not end subject with period
- Body wraps at 72 characters
- Focus on WHAT and WHY, not HOW
- Be specific but concise

Return ONLY the commit message. No explanations, no markdown formatting.`
}
