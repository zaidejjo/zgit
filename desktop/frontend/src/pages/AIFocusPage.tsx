import { useEffect } from "react";
import { useAppStore } from "@/store/app";
import ModeToggle from "@/components/ModeToggle";
import SessionSidebar from "@/components/SessionSidebar";
import ModelSelectorPopover from "@/components/ModelSelectorPopover";
import ChatMessage from "@/components/ChatMessage";
import ProposalCard from "@/components/ProposalCard";
import { Send, RefreshCw, Bot, Loader2, Square, Minimize2, ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useNavigate } from "@tanstack/react-router";
import { useState, useRef } from "react";

export default function AIFocusPage() {
  const navigate = useNavigate();
  const {
    aiMessages, aiProposals, aiThinking, aiError, aiMode, aiSessionActive,
    sendAgentMessage, sendAskMessage, askChatCancel,
    approveProposal, rejectProposal, resetAgentSession,
    toggleAIFullscreen, fetchAIConfig,
  } = useAppStore();

  const [input, setInput] = useState("");
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    fetchAIConfig();
    // Ensure session is active — Ask mode just needs the UI active,
    // Agent mode needs a backend session.
    if (!aiSessionActive) {
      if (aiMode === "agent") {
        useAppStore.getState().startAgentSession();
      } else {
        useAppStore.setState({ aiSessionActive: true });
      }
    }
  }, [fetchAIConfig, aiSessionActive, aiMode]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [aiMessages, aiThinking]);

  useEffect(() => {
    setTimeout(() => inputRef.current?.focus(), 100);
  }, []);

  const handleSend = async () => {
    const msg = input.trim();
    if (!msg || aiThinking) return;
    setInput("");
    if (aiMode === "agent") {
      await sendAgentMessage(msg);
    } else {
      await sendAskMessage(msg);
    }
  };

  const pendingProposals = aiProposals.filter((p) => p.status === "pending");
  const visibleMessageCount = aiMessages.filter((m) => m.role !== "system").length;

  return (
    <div className="h-full flex flex-col bg-background">
      {/* Header */}
      <div className="flex items-center justify-between px-4 h-14 border-b border-border shrink-0">
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate({ to: "/" })}
            className="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
            title="Back to main"
          >
            <ArrowLeft className="w-4 h-4" />
          </button>
          <ModeToggle />
          <ModelSelectorPopover align="left" variant="header" />
        </div>
        <div className="flex items-center gap-1">
          {aiSessionActive && (
            <Button
              variant="ghost"
              size="sm"
              className="h-8 px-2 text-xs gap-1 text-muted-foreground"
              onClick={resetAgentSession}
            >
              <RefreshCw className="w-3.5 h-3.5" />
              Reset
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            className="h-8 px-2 text-xs gap-1 text-muted-foreground"
            onClick={() => {
              toggleAIFullscreen();
              navigate({ to: "/" });
            }}
          >
            <Minimize2 className="w-3.5 h-3.5" />
            Exit Focus
          </Button>
        </div>
      </div>

      {/* Body: Sidebar + Chat */}
      <div className="flex flex-1 min-h-0">
        <SessionSidebar />
        <div className="flex-1 flex flex-col min-w-0">
          {/* Messages area — full height */}
          <div className="flex-1 overflow-y-auto py-4 space-y-1 max-w-4xl mx-auto w-full px-4">
            {!aiSessionActive ? (
              <div className="flex flex-col items-center justify-center h-full text-center">
                <Bot className="w-12 h-12 text-muted-foreground/30 mb-4" />
                <p className="text-base text-muted-foreground/60">
                  {aiMode === "agent"
                    ? "Start an agent session to begin"
                    : "Start asking questions about your repository"}
                </p>
              </div>
            ) : visibleMessageCount === 0 && !aiThinking ? (
              <div className="flex flex-col items-center justify-center h-full text-center">
                <Bot className="w-12 h-12 text-muted-foreground/30 mb-4" />
                <p className="text-base text-muted-foreground/60">
                  {aiMode === "agent"
                    ? "Ask me to help with Git operations, review changes, or resolve conflicts."
                    : "Ask me anything about this repository."}
                </p>
              </div>
            ) : (
              aiMessages.map((msg, i) => {
                const isLastAssistantMsg = msg.role === "assistant" && aiMessages.slice(i + 1).every(m => m.role !== "assistant");
                return (
                  <div key={i}>
                    <ChatMessage
                      message={msg}
                      index={i}
                      isLast={isLastAssistantMsg && i === aiMessages.length - 1}
                      onEdit={(idx, text) => {
                        const msgs = [...useAppStore.getState().aiMessages];
                        msgs[idx] = { ...msgs[idx], content: text };
                        useAppStore.setState({ aiMessages: msgs });
                        if (useAppStore.getState().aiMode === "agent") {
                          useAppStore.getState().sendAgentMessage(text);
                        } else {
                          useAppStore.getState().sendAskMessage(text);
                        }
                      }}
                      onRegenerate={() => {
                        const lastUser = [...useAppStore.getState().aiMessages]
                          .reverse()
                          .find(m => m.role === "user");
                        if (lastUser?.content) {
                          if (useAppStore.getState().aiMode === "agent") {
                            useAppStore.getState().sendAgentMessage(lastUser.content);
                          } else {
                            useAppStore.getState().sendAskMessage(lastUser.content);
                          }
                        }
                      }}
                    />
                    {aiMode === "agent" && msg.role === "assistant" && pendingProposals.length > 0 && (
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
                );
              })
            )}

            {aiThinking && (
              <div className="flex items-center gap-2 px-4 py-2 text-sm text-muted-foreground">
                <Loader2 className="w-4 h-4 animate-spin" />
                <span>{aiMode === "agent" ? "Thinking & analyzing..." : "Thinking..."}</span>
              </div>
            )}

            <div ref={messagesEndRef} />
          </div>

          {/* Error banner */}
          {aiError && (
            <div className="px-6 py-2 bg-destructive/10 border-t border-destructive/20">
              <p className="text-sm text-destructive-foreground">{aiError}</p>
            </div>
          )}

          {/* Input area */}
          <div className="border-t border-border p-4 shrink-0">
            <div className="max-w-4xl mx-auto">
              <div className="flex items-center gap-2">
                <input
                  ref={inputRef}
                  type="text"
                  className="flex-1 h-10 px-4 text-sm rounded-lg border border-border bg-background focus:outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground/50"
                  placeholder={
                    aiThinking
                      ? "Waiting for response..."
                      : aiMode === "agent"
                        ? "Ask AI agent to help with Git..."
                        : "Ask a question about the repo..."
                  }
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && !e.shiftKey) {
                      e.preventDefault();
                      handleSend();
                    }
                  }}
                  disabled={aiThinking}
                />
                {aiThinking ? (
                  <Button
                    size="sm"
                    variant="outline"
                    className="h-10 px-3 shrink-0"
                    onClick={aiMode === "ask" ? askChatCancel : undefined}
                  >
                    <Square className="w-4 h-4" />
                  </Button>
                ) : (
                  <Button
                    size="sm"
                    className="h-10 px-4 shrink-0"
                    onClick={handleSend}
                    disabled={!input.trim()}
                  >
                    <Send className="w-4 h-4 mr-1.5" />
                    Send
                  </Button>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
