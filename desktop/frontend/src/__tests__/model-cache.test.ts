import { describe, it, expect } from "bun:test";

/**
 * Test for merging fetched + cached models (deduplicated).
 * This logic is used in SettingsPage.tsx to display available models.
 */
function mergeModels(fetched: string[], cached: string[]): string[] {
  const seen = new Set<string>();
  const result: string[] = [];
  for (const m of [...fetched, ...cached]) {
    if (!seen.has(m)) {
      seen.add(m);
      result.push(m);
    }
  }
  return result;
}

describe("model cache merge", () => {
  it("deduplicates fetched and cached models", () => {
    const fetched = ["gpt-4o-mini", "gpt-4o", "claude-3-haiku"];
    const cached = ["claude-3-haiku", "claude-sonnet-4", "gemini-flash"];
    const merged = mergeModels(fetched, cached);
    expect(merged).toEqual([
      "gpt-4o-mini",
      "gpt-4o",
      "claude-3-haiku",
      "claude-sonnet-4",
      "gemini-flash",
    ]);
  });

  it("handles empty arrays", () => {
    expect(mergeModels([], [])).toEqual([]);
    expect(mergeModels(["a"], [])).toEqual(["a"]);
    expect(mergeModels([], ["b"])).toEqual(["b"]);
  });

  it("preserves order: fetched first, then cached", () => {
    const fetched = ["model-1", "model-2"];
    const cached = ["model-3", "model-4"];
    expect(mergeModels(fetched, cached)).toEqual([
      "model-1",
      "model-2",
      "model-3",
      "model-4",
    ]);
  });
});
