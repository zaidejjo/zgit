package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Anthropic implements Generator for Anthropic Claude.
type Anthropic struct {
	cfg Config
}

// NewAnthropic creates a new Anthropic provider.
func NewAnthropic(cfg Config) *Anthropic {
	return &Anthropic{cfg: cfg}
}

// anthropicRequest is the request body for Anthropic Messages API.
type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system"`
	Messages    []anthropicMessage `json:"messages"`
	Temperature float64            `json:"temperature"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse is the response from Anthropic Messages API.
type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (a *Anthropic) GenerateCommitMessage(ctx context.Context, diff string, cfg Config) (string, error) {
	return a.GenerateCommitMessageWithContext(ctx, diff, CommitContext{}, cfg)
}

func (a *Anthropic) GenerateCommitMessageWithContext(ctx context.Context, diff string, context CommitContext, cfg Config) (string, error) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoints[ProviderAnthropic]
	}

	model := cfg.Model
	if model == "" {
		model = DefaultModels[ProviderAnthropic]
	}

	// Truncate diff
	truncatedDiff := diff
	if len(truncatedDiff) > 6000 {
		truncatedDiff = truncatedDiff[:6000] + "\n# ... (diff truncated)"
	}

	if strings.TrimSpace(truncatedDiff) == "" {
		// Check if this is an empty repo (no commits at all)
		return "", fmt.Errorf("staged diff is empty — stage some changes first (this repo has no commits yet)")
	}

	// Build context-aware system prompt
	systemPrompt := buildSystemPrompt()
	if context.Branch != "" || len(context.RecentMessages) > 0 {
		var contextLines []string
		if context.Branch != "" {
			contextLines = append(contextLines, fmt.Sprintf("Current branch: %s", context.Branch))
		}
		if context.StatusSummary != "" {
			contextLines = append(contextLines, fmt.Sprintf("Repo status: %s", context.StatusSummary))
		}
		if context.ChangedFiles > 0 {
			contextLines = append(contextLines, fmt.Sprintf("Changed files: %d", context.ChangedFiles))
		}
		if len(context.RecentMessages) > 0 {
			contextLines = append(contextLines, "Recent commit messages (match this style):")
			for _, msg := range context.RecentMessages {
				contextLines = append(contextLines, fmt.Sprintf("  - %s", msg))
			}
		}
		contextBlock := strings.Join(contextLines, "\n")
		systemPrompt = fmt.Sprintf("%s\n\nRepository Context:\n%s", systemPrompt, contextBlock)
	}

	reqBody := anthropicRequest{
		Model:     model,
		MaxTokens: 300,
		System:    systemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: fmt.Sprintf("Generate a commit message for this diff:\n\n```\n%s\n```", truncatedDiff)},
		},
		Temperature: 0.3,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", cfg.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respData))
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(respData, &anthropicResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if anthropicResp.Error != nil {
		return "", fmt.Errorf("API error: %s", anthropicResp.Error.Message)
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("API returned no content")
	}

	msg := strings.TrimSpace(anthropicResp.Content[0].Text)
	return msg, nil
}

// GenerateCommitMessageStream streams tokens via onToken callback for Anthropic Claude.
func (a *Anthropic) GenerateCommitMessageStream(ctx context.Context, diff string, context CommitContext, cfg Config, onToken func(string)) (string, error) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoints[ProviderAnthropic]
	}

	model := cfg.Model
	if model == "" {
		model = DefaultModels[ProviderAnthropic]
	}

	// Truncate diff
	truncatedDiff := diff
	if len(truncatedDiff) > 6000 {
		truncatedDiff = truncatedDiff[:6000] + "\n# ... (diff truncated)"
	}

	if strings.TrimSpace(truncatedDiff) == "" {
		return "", fmt.Errorf("staged diff is empty — stage some changes first (this repo has no commits yet)")
	}

	// Build context-aware system prompt
	systemPrompt := buildSystemPrompt()
	if context.Branch != "" || len(context.RecentMessages) > 0 {
		var contextLines []string
		if context.Branch != "" {
			contextLines = append(contextLines, fmt.Sprintf("Current branch: %s", context.Branch))
		}
		if context.StatusSummary != "" {
			contextLines = append(contextLines, fmt.Sprintf("Repo status: %s", context.StatusSummary))
		}
		if context.ChangedFiles > 0 {
			contextLines = append(contextLines, fmt.Sprintf("Changed files: %d", context.ChangedFiles))
		}
		if len(context.RecentMessages) > 0 {
			contextLines = append(contextLines, "Recent commit messages (match this style):")
			for _, msg := range context.RecentMessages {
				contextLines = append(contextLines, fmt.Sprintf("  - %s", msg))
			}
		}
		contextBlock := strings.Join(contextLines, "\n")
		systemPrompt = fmt.Sprintf("%s\n\nRepository Context:\n%s", systemPrompt, contextBlock)
	}

	// Build user messages (exclude system message — Anthropic uses top-level system param)
	var userMessages []anthropicToolMsg
	userMessages = append(userMessages, anthropicToolMsg{
		Role: "user",
		Content: []anthropicBlock{{
			Type: "text",
			Text: fmt.Sprintf("Generate a commit message for this diff:\n\n```\n%s\n```", truncatedDiff),
		}},
	})

	reqBody := anthropicAskRequest{
		Model:       model,
		MaxTokens:   300,
		System:      systemPrompt,
		Messages:    userMessages,
		Temperature: 0.3,
		Stream:      true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", cfg.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respData, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respData))
	}

	var fullContent string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var delta anthropicStreamDelta
		if err := json.Unmarshal([]byte(data), &delta); err != nil {
			continue
		}
		switch delta.Type {
		case "content_block_delta":
			if delta.Delta != nil && delta.Delta.Text != "" {
				fullContent += delta.Delta.Text
				if onToken != nil {
					onToken(delta.Delta.Text)
				}
			}
		case "message_stop":
			return strings.TrimSpace(fullContent), nil
		case "error":
			return "", fmt.Errorf("Anthropic API error: %s", string(data))
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("stream read error: %w", err)
	}

	return strings.TrimSpace(fullContent), nil
}
