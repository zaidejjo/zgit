import { describe, it, expect } from "bun:test";

/**
 * Test for building AI context summary from repo status.
 * Mirrors the context preview logic in StatusPage.tsx.
 */
function buildContextSummary(model: string, branch: string, status: {
  staged_count: number;
  unstaged_count: number;
  untracked_count: number;
} | null): string {
  const modelName = model || "default model";
  const branchName = branch || "unknown";
  let summary = `AI context: ${modelName} | branch: ${branchName}`;

  if (status) {
    summary += `\n${status.staged_count} staged | ${status.unstaged_count} unstaged | ${status.untracked_count} untracked`;
  }

  return summary;
}

describe("AI context summary", () => {
  it("builds context with model and branch", () => {
    const result = buildContextSummary("claude-sonnet-4", "feature/auth", null);
    expect(result).toContain("claude-sonnet-4");
    expect(result).toContain("feature/auth");
  });

  it("includes status counts when status exists", () => {
    const result = buildContextSummary("gpt-4o-mini", "main", {
      staged_count: 3,
      unstaged_count: 2,
      untracked_count: 1,
    });
    expect(result).toContain("3 staged");
    expect(result).toContain("2 unstaged");
    expect(result).toContain("1 untracked");
  });

  it("handles null status gracefully", () => {
    const result = buildContextSummary("gpt-4o-mini", "main", null);
    expect(result).not.toContain("staged");
    expect(result).not.toContain("unstaged");
  });

  it("uses default model when none set", () => {
    const result = buildContextSummary("", "main", null);
    expect(result).toContain("default model");
  });
});

/**
 * Test for error categorization logic (mirrors frontend parsing).
 */
function categorizeError(message: string): {
  category: string;
  retryable: boolean;
  suggestedModel?: string;
} {
  const lower = message.toLowerCase();
  if (lower.includes("401") || lower.includes("403") || lower.includes("invalid api key") || lower.includes("unauthorized")) {
    return { category: "auth", retryable: false };
  }
  if (lower.includes("429") || lower.includes("rate limit") || lower.includes("too many requests")) {
    return { category: "rate_limit", retryable: true };
  }
  if (lower.includes("context length") || lower.includes("maximum context") || lower.includes("token limit")) {
    return { category: "context_length", retryable: true, suggestedModel: "claude-3-haiku-20240307" };
  }
  if (lower.includes("connection") || lower.includes("timeout") || lower.includes("network")) {
    return { category: "network", retryable: true };
  }
  return { category: "provider", retryable: true };
}

describe("error categorization", () => {
  it("categorizes auth errors", () => {
    const result = categorizeError("401 Invalid API key");
    expect(result.category).toBe("auth");
    expect(result.retryable).toBe(false);
  });

  it("categorizes rate limit errors", () => {
    const result = categorizeError("429 rate limit exceeded");
    expect(result.category).toBe("rate_limit");
    expect(result.retryable).toBe(true);
  });

  it("categorizes context length errors with suggested model", () => {
    const result = categorizeError("context length exceeded");
    expect(result.category).toBe("context_length");
    expect(result.retryable).toBe(true);
    expect(result.suggestedModel).toBe("claude-3-haiku-20240307");
  });

  it("categorizes network errors", () => {
    const result = categorizeError("connection timeout");
    expect(result.category).toBe("network");
    expect(result.retryable).toBe(true);
  });

  it("defaults to provider error", () => {
    const result = categorizeError("unknown error");
    expect(result.category).toBe("provider");
    expect(result.retryable).toBe(true);
  });
});
