package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zaidejjo/zgit/pkg/core/ai"
	gitpkg "github.com/zaidejjo/zgit/pkg/core/git"
)

// aiCommitMsgResult carries the result of AI commit message generation.
type aiCommitMsgResult struct {
	Success bool
	Message string // the generated commit message (subject + optional body)
	Error   string
}

// aiCommitMsgViewID distinguishes AI commit result from regular commit result.
const aiCommitMsgViewID = -6

// generateAICommitMsg reads staged diff, calls AI, populates commit dialog.
// Runs synchronously in a goroutine, sends result via msgs channel.
func (m *Model) generateAICommitMsg() {
	if m.aiData.Provider == "" || m.aiData.APIKey == "" {
		m.msgs <- teaMsg{
			view: aiCommitMsgViewID,
			data: aiCommitMsgResult{
				Success: false,
				Error:   "AI not configured. Set up provider + API key in config (Ctrl+P → Open Config).",
			},
		}
		return
	}

	// Read staged diff with unified text
	ctx, cancel := context.WithTimeout(context.Background(), 10e9)
	defer cancel()

	files, err := m.git.DiffFiles(ctx, gitpkg.DiffOptions{Cached: true, Unified: true})
	if err != nil {
		m.msgs <- teaMsg{
			view: aiCommitMsgViewID,
			data: aiCommitMsgResult{
				Success: false,
				Error:   fmt.Sprintf("Read staged diff: %v", err),
			},
		}
		return
	}

	// Build diff text from per-file unified diffs
	var diffParts []string
	for _, f := range files {
		if f.UnifiedDiff != "" {
			diffParts = append(diffParts, f.UnifiedDiff)
		}
	}
	diffText := strings.Join(diffParts, "\n")

	if strings.TrimSpace(diffText) == "" {
		m.msgs <- teaMsg{
			view: aiCommitMsgViewID,
			data: aiCommitMsgResult{
				Success: false,
				Error:   "No staged changes — stage files first before generating commit message.",
			},
		}
		return
	}

	// Call AI generator
	aiCfg := ai.Config{
		Provider: ai.ProviderKind(m.aiData.Provider),
		APIKey:   m.aiData.APIKey,
		Model:    m.aiData.Model,
		Endpoint: m.aiData.Endpoint,
	}

	genCtx, genCancel := context.WithTimeout(context.Background(), 30e9)
	defer genCancel()

	generator, err := ai.NewGenerator(aiCfg)
	if err != nil {
		m.msgs <- teaMsg{
			view: aiCommitMsgViewID,
			data: aiCommitMsgResult{
				Success: false,
				Error:   fmt.Sprintf("Create AI provider: %v", err),
			},
		}
		return
	}

	start := time.Now()
	msg, err := generator.GenerateCommitMessage(genCtx, diffText, aiCfg)
	if err != nil {
		m.msgs <- teaMsg{
			view: aiCommitMsgViewID,
			data: aiCommitMsgResult{
				Success: false,
				Error:   fmt.Sprintf("AI generation failed: %v", err),
			},
		}
		return
	}

	elapsed := time.Since(start).Round(time.Millisecond)
	m.msgs <- teaMsg{
		view: aiCommitMsgViewID,
		data: aiCommitMsgResult{
			Success: true,
			Message: msg,
			Error:   fmt.Sprintf("Generated in %v", elapsed),
		},
	}
}

// populateCommitFromAI fills the commit dialog Subject and Body from an AI-generated message.
// Parses first line as subject, rest as body.
func (m *Model) populateCommitFromAI(msg string) {
	parts := strings.SplitN(msg, "\n", 2)
	subject := strings.TrimSpace(parts[0])
	body := ""
	if len(parts) > 1 {
		body = strings.TrimSpace(parts[1])
	}

	m.commitDlg.Subject.SetValue(subject)
	if body != "" {
		m.commitDlg.Body.SetValue(body)
	}
}
