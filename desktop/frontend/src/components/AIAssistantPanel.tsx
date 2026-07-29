import { useEffect, useRef, useState } from "react";
import {
  Send, RefreshCw, Bot, PanelRightClose, Loader2, Maximize2, Minimize2, Square,
} from "lucide-react";
import { useAppStore } from "@/store/app";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import ChatMessage from "@/components/ChatMessage";
import ProposalCard from "@/components/ProposalCard";
import ModeToggle from "@/components/ModeToggle";
import SessionSidebar from "@/components/SessionSidebar";
import ModelSelectorPopover from "@/components/ModelSelectorPopover";

const SLASH_HINTS = ["/help", "/context", "/clear", "/model", "/review"];

export default function AIAssistantPanel() {
  const {
    aiPanelOpen, toggleAIPanel,
    aiSessionActive, aiMessages, aiProposals, aiThinking, aiError,
    aiMode, aiFullscreen,
    sendAgentMessage, sendAskMessage, askChatCancel,
    approveProposal, rejectProposal, resetAgentSession,
    toggleAIFullscreen,
    fetchAIConfig,
  } = useAppStore();

  const [input, setInput] = useState("");
  const [showSlashHints, setShowSlashHints] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Refresh config on mount
  useEffect(() => {
    fetchAIConfig();
  }, [fetchAIConfig]);

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

  // Auto-activate Ask mode when panel opens — shows input area immediately
  useEffect(() => {
    if (aiPanelOpen && !aiSessionActive) {
      if (aiMode === "ask") {
        useAppStore.setState({ aiSessionActive: true });
      }
    }
  }, [aiPanelOpen, aiSessionActive, aiMode]);

  const handleSend = async () => {
    const msg = input.trim();
    if (!msg || aiThinking) return;
    setInput("");
    setShowSlashHints(false);
    if (aiMode === "agent") {
      await sendAgentMessage(msg);
    } else {
      await sendAskMessage(msg);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
    if (e.key === "Escape") {
      if (aiFullscreen) {
        toggleAIFullscreen();
      } else {
        toggleAIPanel();
      }
    }
    // Slash command detection
    if (e.key === "/" && input === "") {
      setShowSlashHints(true);
    }
  };

  const handleSlashCommand = (cmd: string) => {
    setShowSlashHints(false);
    switch (cmd) {
      case "/help":
        setInput("What commands are available?");
        break;
      case "/context":
        setInput("Show me the current repository context");
        break;
      case "/clear":
        resetAgentSession();
        return;
      case "/review":
        setInput("Review the current branch changes");
        break;
      default:
        break;
    }
    // Auto-send
    setTimeout(() => {
      const msg = input;
      if (msg.trim()) handleSend();
    }, 50);
  };

  // Filter out system messages for display count
  const visibleMessageCount = aiMessages.filter((m) => m.role !== "system").length;
  const pendingProposals = aiProposals.filter((p) => p.status === "pending");

  const inputPlaceholder = aiThinking
    ? "Waiting for response..."
    : aiMode === "agent"
      ? "Ask AI agent to help with Git..."
      : "Ask a question about the repo...";

  return (
    <>
      {/* Overlay (only in panel mode, not fullscreen) */}
      {aiPanelOpen && !aiFullscreen && (
        <div
          className="fixed inset-0 z-40 command-overlay"
          onClick={toggleAIPanel}
        />
      )}

      {/* Panel */}
      <div
        className={cn(
          aiFullscreen
            ? "fixed inset-0 z-50 bg-background flex flex-col"
            : "fixed top-0 right-0 z-50 h-full w-[520px] max-w-[100vw] glass rounded-none border-r-0 border-t-0 border-b-0 shadow-2xl flex flex-col transition-transform duration-300 ease-in-out",
          aiPanelOpen ? "translate-x-0" : "translate-x-full"
        )}
      >
        {/* ===== Header ===== */}
        <div className="flex items-center justify-between px-3 h-12 border-b border-border shrink-0">
          <div className="flex items-center gap-2 min-w-0">
            {/* Mode toggle (Ask / Agent) */}
            <ModeToggle />

            {/* Model selector */}
            {aiSessionActive && (
              <ModelSelectorPopover align="left" variant="header" />
            )}
          </div>

          <div className="flex items-center gap-1 shrink-0">
            {aiSessionActive && (
              <>
                {/* Reset session */}
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 w-7 p-0"
                  onClick={resetAgentSession}
                  title="Reset session"
                >
                  <RefreshCw className="w-3.5 h-3.5" />
                </Button>
              </>
            )}
            {/* Fullscreen toggle */}
            <Button
              variant="ghost"
              size="sm"
              className="h-7 w-7 p-0"
              onClick={toggleAIFullscreen}
              title={aiFullscreen ? "Exit fullscreen (Esc)" : "Fullscreen focus (Ctrl+Shift+F)"}
            >
              {aiFullscreen ? (
                <Minimize2 className="w-3.5 h-3.5" />
              ) : (
                <Maximize2 className="w-3.5 h-3.5" />
              )}
            </Button>
            {/* Close */}
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

        {/* ===== Body: Sidebar + Content ===== */}
        {!aiSessionActive ? (
          /* Session start screen */
          <div className="flex-1 flex flex-col items-center justify-center px-6 text-center">
            <div className="w-12 h-12 rounded-full bg-primary/10 flex items-center justify-center mb-4">
              <Bot className="w-6 h-6 text-primary" />
            </div>
            <h3 className="text-base font-semibold mb-1">
              {aiMode === "agent" ? "AI Agent Assistant" : "AI Assistant"}
            </h3>
            <p className="text-sm text-muted-foreground max-w-xs mb-3">
              {aiMode === "agent"
                ? "Autonomous Git assistant. Inspects repo state, proposes actions, and waits for your approval."
                : "Ask questions about your repository, Git concepts, or get help with code."}
            </p>
            <p className="text-xs text-muted-foreground/60 mb-6 max-w-xs">
              Configure model and provider in Settings
            </p>
            {aiMode === "agent" && (
              <Button size="sm" onClick={() => useAppStore.getState().startAgentSession()}>
                <Bot className="w-4 h-4 mr-1.5" />
                Start Agent Session
              </Button>
            )}
            <p className="text-[10px] text-muted-foreground/40 mt-6">
              Ctrl+Shift+A to toggle • {aiMode === "agent" ? "Agent" : "Ask"} mode
            </p>
          </div>
        ) : (
          <div className="flex flex-1 min-h-0">
            {/* Session sidebar */}
            <SessionSidebar />

            {/* Main content area */}
            <div className="flex-1 flex flex-col min-w-0">
              {/* Messages area */}
              <div className="flex-1 overflow-y-auto py-3 space-y-1">
                {visibleMessageCount === 0 && !aiThinking && (
                  <div className="flex flex-col items-center justify-center h-full text-center px-6">
                    <Bot className="w-8 h-8 text-muted-foreground/30 mb-3" />
                    <p className="text-sm text-muted-foreground/60">
                      {aiMode === "agent"
                        ? "Ask me to help with Git operations, review changes, or resolve conflicts."
                        : "Ask me anything about this repository or Git."}
                    </p>
                  </div>
                )}

                {aiMessages.map((msg, i) => {
                  const isLastMsg = i === aiMessages.length - 1;
                  const isLastAssistantMsg = msg.role === "assistant" && aiMessages.slice(i + 1).every(m => m.role !== "assistant");
                  return (
                  <div key={i}>
                    <ChatMessage
                      message={msg}
                      index={i}
                      isLast={isLastAssistantMsg && isLastMsg}
                      onEdit={(idx, text) => {
                        // Replace user message and re-send
                        const msgs = [...useAppStore.getState().aiMessages];
                        msgs[idx] = { ...msgs[idx], content: text };
                        useAppStore.setState({ aiMessages: msgs });
                        if (useAppStore.getState().aiMode === "agent") {
                          useAppStore.getState().sendAgentMessage(text);
                        } else {
                          useAppStore.getState().sendAskMessage(text);
                        }
                      }}
                      onRegenerate={(idx) => {
                        // Re-send the last user message
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
                    {/* Show proposals after the assistant message they belong to (agent mode only) */}
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
                })}

                {/* Thinking indicator */}
                {aiThinking && (
                  <div className="flex items-center gap-2 px-4 py-2 text-xs text-muted-foreground">
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    <span>{aiMode === "agent" ? "Thinking & analyzing..." : "Thinking..."}</span>
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
                {/* Slash command hints */}
                {showSlashHints && input === "" && (
                  <div className="mb-2 rounded-lg border border-border bg-popover shadow-sm overflow-hidden">
                    {SLASH_HINTS.map((cmd) => (
                      <button
                        key={cmd}
                        onClick={() => handleSlashCommand(cmd)}
                        className="w-full flex items-center gap-2 px-3 py-1.5 text-xs text-left hover:bg-muted transition-colors"
                      >
                        <code className="text-primary font-mono">{cmd}</code>
                        <span className="text-muted-foreground">
                          {cmd === "/help" && "Show available commands"}
                          {cmd === "/context" && "Show repo context"}
                          {cmd === "/clear" && "Clear conversation"}
                          {cmd === "/model" && "Switch AI model"}
                          {cmd === "/review" && "Review branch diff"}
                        </span>
                      </button>
                    ))}
                  </div>
                )}

                <div className="flex items-center gap-2">
                  <input
                    ref={inputRef}
                    type="text"
                    className="flex-1 h-9 px-3 text-sm rounded-lg border border-border bg-background focus:outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground/50"
                    placeholder={inputPlaceholder}
                    value={input}
                    onChange={(e) => {
                      setInput(e.target.value);
                      if (e.target.value === "") setShowSlashHints(false);
                    }}
                    onKeyDown={handleKeyDown}
                    disabled={aiThinking}
                  />
                  {aiThinking ? (
                    <Button
                      size="sm"
                      variant="outline"
                      className="h-9 px-2 shrink-0"
                      onClick={aiMode === "ask" ? askChatCancel : undefined}
                      title="Cancel"
                    >
                      <Square className="w-3.5 h-3.5" />
                    </Button>
                  ) : (
                    <Button
                      size="sm"
                      className="h-9 w-9 p-0 shrink-0"
                      onClick={handleSend}
                      disabled={!input.trim()}
                    >
                      <Send className="w-4 h-4" />
                    </Button>
                  )}
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </>
  );
}
