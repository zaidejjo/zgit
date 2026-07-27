import { useEffect, useRef, useState } from "react";
import {
  Sparkles, Send, RefreshCw, Bot, PanelRightClose, Loader2, ChevronDown,
} from "lucide-react";
import { useAppStore } from "@/store/app";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import ChatMessage from "@/components/ChatMessage";
import ProposalCard from "@/components/ProposalCard";

// Quick-pick models for the in-header selector
const QUICK_MODELS: { label: string; model: string }[] = [
  { label: "GPT-4o mini", model: "gpt-4o-mini" },
  { label: "GPT-4o", model: "gpt-4o" },
  { label: "Claude 3.5 Sonnet", model: "anthropic/claude-3.5-sonnet" },
  { label: "Claude Sonnet 4", model: "claude-sonnet-4-20250514" },
  { label: "DeepSeek Chat", model: "deepseek-chat" },
  { label: "DeepSeek R1", model: "deepseek/deepseek-r1" },
  { label: "Gemini Flash", model: "google/gemini-2.0-flash-exp" },
  { label: "DeepSeek V4", model: "deepseek/deepseek-v4-flash:free" },
];

export default function AIAssistantPanel() {
  const {
    aiPanelOpen, toggleAIPanel,
    aiSessionActive, aiMessages, aiProposals, aiThinking, aiError,
    sendAgentMessage, approveProposal, rejectProposal, resetAgentSession,
    aiConfig, fetchAIConfig, setAIConfigAction,
  } = useAppStore();

  const [input, setInput] = useState("");
  const [modelOpen, setModelOpen] = useState(false);
  const [editModel, setEditModel] = useState("");
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const modelRef = useRef<HTMLDivElement>(null);

  const currentModel = aiConfig?.model || "";

  // Refresh config on mount
  useEffect(() => {
    fetchAIConfig();
  }, [fetchAIConfig]);

  // Sync editModel when dropdown opens
  useEffect(() => {
    if (modelOpen) {
      setEditModel(currentModel);
    }
  }, [modelOpen, currentModel]);

  // Auto-scroll to bottom on new messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [aiMessages, aiThinking]);

  // Focus input when panel opens
  useEffect(() => {
    if (aiPanelOpen) {
      setTimeout(() => inputRef.current?.focus(), 100);
    }
  }, [aiPanelOpen]);

  // Close model dropdown on outside click
  useEffect(() => {
    if (!modelOpen) return;
    const handler = (e: MouseEvent) => {
      if (modelRef.current && !modelRef.current.contains(e.target as Node)) {
        setModelOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [modelOpen]);

  const handleSend = async () => {
    const msg = input.trim();
    if (!msg || aiThinking) return;
    setInput("");
    await sendAgentMessage(msg);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
    if (e.key === "Escape") {
      toggleAIPanel();
    }
  };

  const handleModelChange = async (model: string) => {
    setEditModel(model);
    setModelOpen(false);
    if (model && aiConfig) {
      await setAIConfigAction(
        aiConfig.provider,
        aiConfig.api_key,
        model,
        aiConfig.endpoint || "",
      );
    }
  };

  const handleModelInputKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      handleModelChange(editModel);
    }
    if (e.key === "Escape") {
      setModelOpen(false);
    }
  };

  // Filter out system messages
  const visibleMessageCount = aiMessages.filter((m) => m.role !== "system").length;
  const pendingProposals = aiProposals.filter((p) => p.status === "pending");

  const providerName = aiConfig?.provider || "";

  return (
    <>
      {/* Overlay */}
      {aiPanelOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/20 backdrop-blur-[1px]"
          onClick={toggleAIPanel}
        />
      )}

      {/* Panel */}
      <div
        className={cn(
          "fixed top-0 right-0 z-50 h-full w-[440px] max-w-[100vw] bg-background border-l border-border shadow-2xl flex flex-col transition-transform duration-300 ease-in-out",
          aiPanelOpen ? "translate-x-0" : "translate-x-full"
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 h-12 border-b border-border shrink-0">
          <div className="flex items-center gap-2 min-w-0">
            <Sparkles className="w-4 h-4 text-primary shrink-0" />
            <span className="text-sm font-semibold shrink-0">AI</span>

            {/* Inline model selector */}
            {aiSessionActive && (
              <div className="relative" ref={modelRef}>
                <button
                  onClick={() => setModelOpen(!modelOpen)}
                  className="flex items-center gap-1 text-[11px] font-mono px-1.5 py-0.5 rounded bg-muted/60 border border-border/50 hover:bg-muted transition-colors max-w-[180px]"
                  title={`Model: ${currentModel || "not set"}`}
                >
                  <span className="truncate">{currentModel || "model"}</span>
                  <ChevronDown className="w-3 h-3 shrink-0 text-muted-foreground" />
                </button>

                {modelOpen && (
                  <div className="absolute left-0 top-full mt-1 z-50 w-72 rounded-lg border bg-popover shadow-lg p-2">
                    {/* Text input for custom model */}
                    <input
                      type="text"
                      value={editModel}
                      onChange={(e) => setEditModel(e.target.value)}
                      onKeyDown={handleModelInputKeyDown}
                      className="w-full h-8 px-2 text-xs font-mono rounded border border-border bg-background focus:outline-none focus:ring-1 focus:ring-ring mb-2"
                      placeholder={providerName ? `e.g. ${providerName}/model-name` : "Type any model..."}
                      autoFocus
                    />

                    {/* Quick-pick chips */}
                    <div className="flex flex-wrap gap-1 mb-1.5">
                      {QUICK_MODELS.map((qm) => (
                        <button
                          key={qm.model}
                          onClick={() => handleModelChange(qm.model)}
                          className={cn(
                            "text-[10px] px-2 py-0.5 rounded-full border font-mono transition-colors",
                            editModel === qm.model
                              ? "border-primary bg-primary/10 text-primary"
                              : "border-border text-muted-foreground/60 hover:border-primary/50 hover:text-foreground"
                          )}
                        >
                          {qm.label}
                        </button>
                      ))}
                    </div>
                    <p className="text-[9px] text-muted-foreground/40 mt-1">
                      Press Enter to confirm • Esc to close
                    </p>
                  </div>
                )}
              </div>
            )}
          </div>

          <div className="flex items-center gap-1 shrink-0">
            {aiSessionActive && (
              <Button
                variant="ghost"
                size="sm"
                className="h-7 w-7 p-0"
                onClick={resetAgentSession}
                title="Reset session"
              >
                <RefreshCw className="w-3.5 h-3.5" />
              </Button>
            )}
            <Button
              variant="ghost"
              size="sm"
              className="h-7 w-7 p-0"
              onClick={toggleAIPanel}
              title="Close (Esc)"
            >
              <PanelRightClose className="w-4 h-4" />
            </Button>
          </div>
        </div>

        {!aiSessionActive ? (
          /* Session start screen */
          <div className="flex-1 flex flex-col items-center justify-center px-6 text-center">
            <div className="w-12 h-12 rounded-full bg-primary/10 flex items-center justify-center mb-4">
              <Bot className="w-6 h-6 text-primary" />
            </div>
            <h3 className="text-base font-semibold mb-1">AI Agent Assistant</h3>
            <p className="text-sm text-muted-foreground max-w-xs mb-3">
              Autonomous Git assistant powered by{" "}
              {(() => {
                const cfg = useAppStore.getState().aiConfig;
                const model = cfg?.model ? ` (${cfg.model})` : "";
                return (cfg?.provider || "AI") + model;
              })()}
              . Inspects repo state, proposes actions, and waits for your approval.
            </p>
            <p className="text-xs text-muted-foreground/60 mb-6 max-w-xs">
              Configure model and provider in Settings
            </p>
            <p className="text-[10px] text-muted-foreground/40">
              Ctrl+Shift+A to toggle • Esc to close
            </p>
          </div>
        ) : (
          <>
            {/* Messages area */}
            <div className="flex-1 overflow-y-auto py-3 space-y-1">
              {visibleMessageCount === 0 && !aiThinking && (
                <div className="flex flex-col items-center justify-center h-full text-center px-6">
                  <Bot className="w-8 h-8 text-muted-foreground/30 mb-3" />
                  <p className="text-sm text-muted-foreground/60">
                    Ask me to help with Git operations, review changes, or resolve conflicts.
                  </p>
                </div>
              )}

              {aiMessages.map((msg, i) => (
                <div key={i}>
                  <ChatMessage message={msg} />
                  {/* Show proposals after the assistant message they belong to */}
                  {msg.role === "assistant" && pendingProposals.length > 0 && (
                    <div className="px-4 pb-2 space-y-2">
                      {pendingProposals.map((prop) => (
                        <ProposalCard
                          key={prop.id}
                          proposal={prop}
                          onApprove={approveProposal}
                          onReject={rejectProposal}
                          disabled={aiThinking}
                        />
                      ))}
                    </div>
                  )}
                </div>
              ))}

              {/* Thinking indicator */}
              {aiThinking && (
                <div className="flex items-center gap-2 px-4 py-2 text-xs text-muted-foreground">
                  <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  <span>Thinking...</span>
                </div>
              )}

              <div ref={messagesEndRef} />
            </div>

            {/* Error banner */}
            {aiError && (
              <div className="px-4 py-2 bg-destructive/10 border-t border-destructive/20">
                <p className="text-xs text-destructive-foreground">{aiError}</p>
              </div>
            )}

            {/* Input area */}
            <div className="border-t border-border p-3 shrink-0">
              <div className="flex items-center gap-2">
                <input
                  ref={inputRef}
                  type="text"
                  className="flex-1 h-9 px-3 text-sm rounded-lg border border-border bg-background focus:outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground/50"
                  placeholder={aiThinking ? "Waiting for response..." : "Ask the AI agent..."}
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyDown={handleKeyDown}
                  disabled={aiThinking}
                />
                <Button
                  size="sm"
                  className="h-9 w-9 p-0 shrink-0"
                  onClick={handleSend}
                  disabled={!input.trim() || aiThinking}
                >
                  {aiThinking ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <Send className="w-4 h-4" />
                  )}
                </Button>
              </div>
            </div>
          </>
        )}
      </div>
    </>
  );
}
