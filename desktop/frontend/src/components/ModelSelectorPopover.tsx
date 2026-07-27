import { useRef, useState, useEffect, useCallback } from "react";
import { useAppStore } from "@/store/app";
import { cn } from "@/lib/utils";
import { ChevronDown, Sparkles, Search } from "lucide-react";

// Provider categories with recommended models (user-facing labels + API model slugs)
const PROVIDER_CATEGORIES = [
  {
    id: "openrouter",
    label: "OpenRouter",
    models: [
      { label: "GPT-4o Mini", model: "openai/gpt-4o-mini" },
      { label: "GPT-4o", model: "openai/gpt-4o" },
      { label: "Claude 3.5 Sonnet", model: "anthropic/claude-3.5-sonnet" },
      { label: "Claude Sonnet 4", model: "claude-sonnet-4-20250514" },
      { label: "DeepSeek V3", model: "deepseek/deepseek-chat" },
      { label: "DeepSeek R1", model: "deepseek/deepseek-r1" },
      { label: "DeepSeek V4 Flash", model: "deepseek/deepseek-v4-flash" },
      { label: "Gemini Flash", model: "google/gemini-2.0-flash-exp" },
      { label: "Gemma 3", model: "google/gemma-3-27b-it" },
    ],
  },
  {
    id: "opencode",
    label: "OpenCode",
    models: [
      { label: "DeepSeek V4 Flash (Go)", model: "deepseek/deepseek-v4-flash" },
      { label: "DeepSeek V3 (Zen)", model: "deepseek/deepseek-chat" },
      { label: "Claude Sonnet 4", model: "claude-sonnet-4-20250514" },
    ],
  },
  {
    id: "openai",
    label: "OpenAI",
    models: [
      { label: "GPT-4o Mini", model: "gpt-4o-mini" },
      { label: "GPT-4o", model: "gpt-4o" },
      { label: "GPT-4 Turbo", model: "gpt-4-turbo" },
      { label: "GPT-3.5 Turbo", model: "gpt-3.5-turbo" },
    ],
  },
  {
    id: "anthropic",
    label: "Anthropic",
    models: [
      { label: "Claude Sonnet 4", model: "claude-sonnet-4-20250514" },
      { label: "Claude 3.5 Sonnet", model: "claude-3-5-sonnet-20241022" },
      { label: "Claude 3 Opus", model: "claude-3-opus-20240229" },
      { label: "Claude 3 Haiku", model: "claude-3-haiku-20240307" },
    ],
  },
  {
    id: "deepseek",
    label: "DeepSeek",
    models: [
      { label: "DeepSeek Chat (V3)", model: "deepseek-chat" },
      { label: "DeepSeek Coder", model: "deepseek-coder" },
      { label: "DeepSeek R1", model: "deepseek-reasoner" },
    ],
  },
  {
    id: "custom",
    label: "Custom",
    models: [
      { label: "Llama 3.2", model: "llama3.2" },
      { label: "Qwen 2.5 Coder", model: "qwen2.5-coder" },
      { label: "Mistral", model: "mistral" },
      { label: "Codestral", model: "codestral" },
      { label: "DeepSeek V4 Flash", model: "deepseek/deepseek-v4-flash" },
    ],
  },
];

interface ModelSelectorPopoverProps {
  align?: "left" | "right";
  variant?: "header" | "settings";
}

export default function ModelSelectorPopover({
  align = "left",
  variant = "header",
}: ModelSelectorPopoverProps) {
  const { aiConfig, fetchAIConfig, setAIConfigAction } = useAppStore();
  const [open, setOpen] = useState(false);
  const [activeTab, setActiveTab] = useState("openrouter");
  const [customInput, setCustomInput] = useState("");
  const ref = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const currentModel = aiConfig?.model || "";
  const currentProvider = aiConfig?.provider || "";

  useEffect(() => {
    fetchAIConfig();
  }, [fetchAIConfig]);

  // Focus input on open
  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }, [open]);

  // Close on outside click
  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  const handleSelectModel = useCallback(async (provider: string, model: string) => {
    setOpen(false);
    await setAIConfigAction(
      provider,
      aiConfig?.api_key || "",
      model,
      aiConfig?.endpoint || "",
    );
  }, [aiConfig, setAIConfigAction]);

  const handleCustomSubmit = useCallback(async () => {
    const model = customInput.trim();
    if (!model) return;
    setOpen(false);
    setCustomInput("");
    // Determine provider from active tab
    const provider = activeTab === "custom" ? "custom" : activeTab;
    await setAIConfigAction(
      provider,
      aiConfig?.api_key || "",
      model,
      aiConfig?.endpoint || "",
    );
  }, [customInput, activeTab, aiConfig, setAIConfigAction]);

  // Find current category for the selected model
  const currentCategory = PROVIDER_CATEGORIES.find((cat) =>
    cat.models.some((m) => m.model === currentModel)
  );

  return (
    <div className="relative" ref={ref}>
      {/* Trigger button */}
      <button
        onClick={() => setOpen(!open)}
        className={cn(
          "flex items-center gap-1 text-[11px] font-mono rounded border transition-colors",
          variant === "header"
            ? "px-1.5 py-0.5 bg-muted/60 border-border/50 hover:bg-muted max-w-[180px]"
            : "px-2 py-1 bg-background border-input hover:bg-accent max-w-[240px]",
        )}
        title={`Model: ${currentModel || "not set"}`}
      >
        <Sparkles className="w-3 h-3 shrink-0 text-muted-foreground" />
        <span className="truncate">{currentModel || "Select model"}</span>
        <ChevronDown className="w-3 h-3 shrink-0 text-muted-foreground" />
      </button>

      {/* Popover */}
      {open && (
        <div
          className={cn(
            "absolute top-full mt-1 z-50 w-80 rounded-lg border bg-popover shadow-lg",
            align === "right" ? "right-0" : "left-0",
          )}
        >
          {/* Provider tabs */}
          <div className="flex flex-wrap gap-0.5 p-2 border-b border-border">
            {PROVIDER_CATEGORIES.map((cat) => (
              <button
                key={cat.id}
                onClick={() => {
                  setActiveTab(cat.id);
                  // Auto-fill first model when switching tabs
                  if (cat.models.length > 0) {
                    setCustomInput(cat.models[0].model);
                  }
                }}
                className={cn(
                  "text-[10px] px-2 py-1 rounded font-medium transition-colors",
                  activeTab === cat.id
                    ? "bg-primary/10 text-primary"
                    : "text-muted-foreground hover:text-foreground hover:bg-muted"
                )}
              >
                {cat.label}
              </button>
            ))}
          </div>

          {/* Custom model input */}
          <div className="p-2 border-b border-border">
            <div className="flex items-center gap-1">
              <Search className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
              <input
                ref={inputRef}
                type="text"
                value={customInput}
                onChange={(e) => setCustomInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleCustomSubmit();
                  if (e.key === "Escape") setOpen(false);
                }}
                className="flex-1 h-7 px-2 text-xs font-mono rounded border border-border bg-background focus:outline-none focus:ring-1 focus:ring-ring"
                placeholder="Type any model..."
              />
            </div>
          </div>

          {/* Recommended models for active tab */}
          <div className="max-h-52 overflow-y-auto p-2">
            <p className="text-[9px] text-muted-foreground/50 mb-1.5 px-1 uppercase tracking-wider">
              {PROVIDER_CATEGORIES.find((c) => c.id === activeTab)?.label || "Recommended"}
            </p>
            <div className="space-y-0.5">
              {(PROVIDER_CATEGORIES.find((c) => c.id === activeTab)?.models || []).map((m) => {
                const isActive = currentModel === m.model;
                return (
                  <button
                    key={m.model}
                    onClick={() => handleSelectModel(activeTab, m.model)}
                    className={cn(
                      "w-full flex items-center gap-2 px-2 py-1.5 rounded-md text-left text-xs transition-colors",
                      isActive
                        ? "bg-primary/10 text-primary"
                        : "text-foreground hover:bg-muted"
                    )}
                  >
                    <span className={cn(
                      "w-1.5 h-1.5 rounded-full shrink-0",
                      isActive ? "bg-primary" : "bg-muted-foreground/30"
                    )} />
                    <div className="flex-1 min-w-0">
                      <span className="block truncate font-medium">{m.label}</span>
                      <span className="block truncate text-[10px] text-muted-foreground/60 font-mono">
                        {m.model}
                      </span>
                    </div>
                    {isActive && (
                      <span className="text-[9px] text-primary font-medium shrink-0">Active</span>
                    )}
                  </button>
                );
              })}
            </div>
          </div>

          {/* Footer */}
          <div className="px-3 py-1.5 border-t border-border flex items-center justify-between">
            <span className="text-[9px] text-muted-foreground/40">
              {currentProvider ? `Provider: ${currentProvider}` : "No provider set"}
            </span>
            <span className="text-[9px] text-muted-foreground/40">
              Enter to confirm
            </span>
          </div>
        </div>
      )}
    </div>
  );
}
