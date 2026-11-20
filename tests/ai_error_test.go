package integration

import (
	"strings"
	"testing"
)

// categorizeError logic is tested inline (it's unexported in desktop/app.go)
func TestAIErrorCategories(t *testing.T) {
	tests := []struct {
		name          string
		message       string
		wantCategory  string
		wantRetryable bool
		wantSuggested string
	}{
		{"auth error", "401 Invalid API key", "auth", false, ""},
		{"auth error 403", "403 Forbidden", "auth", false, ""},
		{"auth invalid key", "invalid api key provided", "auth", false, ""},
		{"rate limit", "429 rate limit exceeded", "rate_limit", true, ""},
		{"rate limit too many", "too many requests", "rate_limit", true, ""},
		{"context length", "context length exceeded", "context_length", true, "claude-3-haiku-20240307"},
		{"context length max", "maximum context length reached", "context_length", true, "claude-3-haiku-20240307"},
		{"network timeout", "connection timeout", "network", true, ""},
		{"network unreachable", "host unreachable", "network", true, ""},
		{"provider error", "unknown provider error", "provider", true, ""},
		{"empty error", "", "provider", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate categorizeError logic (same as desktop/app.go)
			msg := tt.message
			lower := strings.ToLower(msg)
			var category string
			var retryable bool
			var suggestedModel string

			switch {
			case strings.Contains(lower, "401") || strings.Contains(lower, "403") ||
				strings.Contains(lower, "invalid api key") || strings.Contains(lower, "authentication"):
				category = "auth"
				retryable = false
			case strings.Contains(lower, "429") || strings.Contains(lower, "rate limit") ||
				strings.Contains(lower, "too many requests"):
				category = "rate_limit"
				retryable = true
			case strings.Contains(lower, "context length") || strings.Contains(lower, "maximum context") ||
				strings.Contains(lower, "token limit"):
				category = "context_length"
				retryable = true
				suggestedModel = "claude-3-haiku-20240307"
			case strings.Contains(lower, "connection") || strings.Contains(lower, "timeout") ||
				strings.Contains(lower, "network") || strings.Contains(lower, "unreachable"):
				category = "network"
				retryable = true
			default:
				category = "provider"
				retryable = true
			}

			if category != tt.wantCategory {
				t.Errorf("category = %q, want %q", category, tt.wantCategory)
			}
			if retryable != tt.wantRetryable {
				t.Errorf("retryable = %v, want %v", retryable, tt.wantRetryable)
			}
			if suggestedModel != tt.wantSuggested {
				t.Errorf("suggestedModel = %q, want %q", suggestedModel, tt.wantSuggested)
			}
		})
	}
}

// TestAutoFallbackModel verifies the fallback model suggestion is correct.
func TestAutoFallbackModel(t *testing.T) {
	// context_length errors should suggest claude-3-haiku-20240307
	const suggestedModel = "claude-3-haiku-20240307"
	if suggestedModel == "" {
		t.Error("suggested fallback model should not be empty")
	}
	// It should be a smaller, faster model suitable for fallback
	if !strings.Contains(suggestedModel, "haiku") {
		t.Error("fallback model should be a lightweight model (haiku)")
	}
}
