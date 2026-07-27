import { useEffect, useRef, useState } from "react";
import {
  Sparkles, X, Send, RefreshCw, Bot, PanelRightClose, Loader2,
} from "lucide-react";
import { useAppStore } from "@/store/app";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import ChatMessage from "@/components/ChatMessage";
import ProposalCard from "@/components/ProposalCard";

export default function AIAssistantPanel() {
  const {
    aiPanelOpen, toggleAIPanel,
    aiSessionActive, aiMessages, aiProposals, aiThinking, aiError,
    sendAgentMessage, approveProposal, rejectProposal, resetAgentSession,
  } = useAppStore();

  const [input, setInput] = useState("");
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

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

  // Filter out system and tool messages for count display
  const visibleMessageCount = aiMessages.filter((m) => m.role !== "system").length;

  // Group assistant messages with their subsequent proposals
  const pendingProposals = aiProposals.filter((p) => p.status === "pending");

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
          <div className="flex items-center gap-2">
            <Sparkles className="w-4 h-4 text-primary" />
            <span className="text-sm font-semibold">AI Assistant</span>
            {aiSessionActive && (
              <span className="text-[10px] px-1.5 py-0.5 rounded bg-primary/10 text-primary font-medium uppercase">
                Agentic
              </span>
            )}
          </div>
          <div className="flex items-center gap-1">
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
