import { type ReactNode, useState } from "react";
import { cn } from "@/lib/utils";
import { AgentMessage, AgentToolCall } from "@/store/app";
import { Copy, PencilLine, RefreshCw, Check, X } from "lucide-react";

interface ChatMessageProps {
  message: AgentMessage;
  index?: number;
  onEdit?: (index: number, text: string) => void;
  onRegenerate?: (index: number) => void;
  isLast?: boolean;
}

export default function ChatMessage({ message, index, onEdit, onRegenerate, isLast }: ChatMessageProps) {
  const isUser = message.role === "user";
  const isTool = message.role === "tool";
  const isAssistant = message.role === "assistant";

  const [copied, setCopied] = useState(false);
  const [editing, setEditing] = useState(false);
  const [editText, setEditText] = useState(message.content || "");

  const handleCopy = async () => {
    if (message.content) {
      await navigator.clipboard.writeText(message.content);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    }
  };

  const handleEditSave = () => {
    if (editText.trim() && onEdit && index !== undefined) {
      onEdit(index, editText.trim());
    }
    setEditing(false);
  };

  const handleEditCancel = () => {
    setEditText(message.content || "");
    setEditing(false);
  };

  // Tool result messages: show inline status
  if (isTool) {
    const isError = message.content?.startsWith("❌");
    return (
      <div className="flex justify-start px-4 py-1">
        <div
          className={cn(
            "text-xs font-mono px-2.5 py-1 rounded border max-w-[90%]",
            isError
              ? "bg-destructive/10 border-destructive/20 text-destructive-foreground/80"
              : "bg-primary/5 border-primary/10 text-muted-foreground"
          )}
        >
          {message.content}
        </div>
      </div>
    );
  }

  return (
    <div className={cn("group flex px-4 py-1.5", isUser ? "justify-end" : "justify-start")}>
      <div className="relative max-w-[85%]">
        <div
          className={cn(
            "rounded-lg px-3.5 py-2 space-y-1.5",
            isUser
              ? "bg-primary text-primary-foreground rounded-br-sm"
              : "bg-muted/60 text-foreground rounded-bl-sm border border-border/40"
          )}
        >
          {/* Tool call badges */}
          {message.tool_calls && message.tool_calls.length > 0 && (
            <div className="flex flex-wrap gap-1.5 mb-1.5">
              {message.tool_calls.map((tc) => (
                <ToolCallBadge key={tc.id} toolCall={tc} />
              ))}
            </div>
          )}

          {/* Editing mode (user messages only) */}
          {editing ? (
            <div className="space-y-1.5">
              <textarea
                value={editText}
                onChange={(e) => setEditText(e.target.value)}
                className="w-full min-h-[60px] text-sm rounded border border-border bg-background text-foreground p-2 font-mono focus:outline-none focus:ring-1 focus:ring-ring resize-none"
                autoFocus
              />
              <div className="flex items-center gap-1 justify-end">
                <button
                  onClick={handleEditSave}
                  className="flex items-center gap-1 text-[10px] px-2 py-0.5 rounded bg-primary text-primary-foreground hover:bg-primary/90"
                >
                  <Check className="w-3 h-3" />
                  Save
                </button>
                <button
                  onClick={handleEditCancel}
                  className="flex items-center gap-1 text-[10px] px-2 py-0.5 rounded bg-muted text-muted-foreground hover:text-foreground"
                >
                  <X className="w-3 h-3" />
                  Cancel
                </button>
              </div>
            </div>
          ) : (
            <>
              {message.content ? (
                <MarkdownContent text={message.content} isUser={isUser} />
              ) : (
                message.tool_calls && message.tool_calls.length > 0 && (
                  <p className="text-xs text-muted-foreground italic">
                    Using tools...
                  </p>
                )
              )}
            </>
          )}
        </div>

        {/* Action buttons (visible on hover) */}
        {!editing && (isUser || isAssistant) && (
          <div
            className={cn(
              "absolute -bottom-4 hidden group-hover:flex items-center gap-0.5",
              isUser ? "right-1" : "left-1",
            )}
          >
            {/* Copy - on assistant messages */}
            {isAssistant && message.content && (
              <button
                onClick={handleCopy}
                className="p-0.5 rounded text-muted-foreground/50 hover:text-foreground transition-colors"
                title={copied ? "Copied!" : "Copy response"}
              >
                {copied ? (
                  <Check className="w-3 h-3 text-green-500" />
                ) : (
                  <Copy className="w-3 h-3" />
                )}
              </button>
            )}

            {/* Edit - on user messages */}
            {isUser && onEdit && index !== undefined && (
              <button
                onClick={() => { setEditing(true); setEditText(message.content || ""); }}
                className="p-0.5 rounded text-muted-foreground/50 hover:text-foreground transition-colors"
                title="Edit message"
              >
                <PencilLine className="w-3 h-3" />
              </button>
            )}

            {/* Regenerate - on last assistant message */}
            {isAssistant && isLast && onRegenerate && index !== undefined && (
              <button
                onClick={() => onRegenerate(index)}
                className="p-0.5 rounded text-muted-foreground/50 hover:text-foreground transition-colors"
                title="Regenerate response"
              >
                <RefreshCw className="w-3 h-3" />
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function ToolCallBadge({ toolCall }: { toolCall: AgentToolCall }) {
  const icon = toolCall.name.includes("context") ? "📋" :
    toolCall.name.includes("review") ? "🔍" :
    toolCall.name.includes("conflict") ? "🤝" :
    toolCall.name.includes("suggest") || toolCall.name.includes("command") ? "⚡" : "🔧";

  const label = toolCall.name.replace(/_/g, " ");
  return (
    <span className="inline-flex items-center gap-1 text-[10px] font-medium px-1.5 py-0.5 rounded bg-primary/10 text-primary border border-primary/20">
      <span>{icon}</span>
      <span>{label}</span>
    </span>
  );
}

// --- Lightweight Markdown Renderer ---

function MarkdownContent({ text, isUser }: { text: string; isUser: boolean }) {
  const segments = parseMarkdown(text);
  return (
    <div className={cn("text-sm leading-relaxed space-y-1.5", isUser ? "prose-invert" : "")}>
      {segments.map((seg, i) => (
        <div key={i}>{seg}</div>
      ))}
    </div>
  );
}

type Segment =
  | { type: "paragraph"; content: ReactNode[] }
  | { type: "code"; language: string; code: string }
  | { type: "list"; items: string[] };

function parseMarkdown(text: string): ReactNode[] {
  const nodes: ReactNode[] = [];
  const lines = text.split("\n");
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];

    // Code block
    if (line.trimStart().startsWith("```")) {
      const lang = line.trim().slice(3).trim();
      const codeLines: string[] = [];
      i++;
      while (i < lines.length && !lines[i].trimStart().startsWith("```")) {
        codeLines.push(lines[i]);
        i++;
      }
      i++; // skip closing ```
      nodes.push(
        <pre
          key={`code-${i}`}
          className="bg-muted/30 rounded-md p-2.5 text-xs font-mono overflow-x-auto border border-border/30 my-1"
        >
          {lang && (
            <div className="text-[10px] text-muted-foreground/60 uppercase tracking-wider mb-1 -mt-0.5">
              {lang}
            </div>
          )}
          <code>{escapeHtml(codeLines.join("\n"))}</code>
        </pre>
      );
      continue;
    }

    // List item
    if (line.trimStart().match(/^[-*+]\s/) || line.trimStart().match(/^\d+[.)]\s/)) {
      const items: string[] = [];
      while (i < lines.length) {
        const trimmed = lines[i].trimStart();
        if (trimmed.match(/^[-*+]\s/) || trimmed.match(/^\d+[.)]\s/)) {
          items.push(trimmed.replace(/^[-*+]\s/, "").replace(/^\d+[.)]\s/, ""));
          i++;
        } else if (trimmed === "") {
          i++;
          break;
        } else {
          break;
        }
      }
      nodes.push(
        <ul key={`list-${i}`} className="list-disc list-inside space-y-0.5 text-sm">
          {items.map((item, j) => (
            <li key={j}>
              {renderInlineContent(item)}
            </li>
          ))}
        </ul>
      );
      continue;
    }

    // Blank line
    if (line.trim() === "") {
      i++;
      continue;
    }

    // Regular paragraph — collect consecutive non-blank, non-code lines
    const paraLines: string[] = [];
    while (i < lines.length) {
      const l = lines[i];
      if (l.trim() === "" || l.trimStart().startsWith("```") || l.trimStart().match(/^[-*+]\s/) || l.trimStart().match(/^\d+[.)]\s/)) {
        break;
      }
      paraLines.push(l);
      i++;
    }
    if (paraLines.length > 0) {
      nodes.push(
        <p key={`para-${i}`} className="text-sm">
          {renderInlineContent(paraLines.join("\n"))}
        </p>
      );
    }
  }

  return nodes;
}

function renderInlineContent(text: string): ReactNode[] {
  const nodes: ReactNode[] = [];
  // Split on inline code, bold, italic
  const parts = text.split(/(`[^`]+`)/g);
  let key = 0;

  for (const part of parts) {
    if (part.startsWith("`") && part.endsWith("`")) {
      nodes.push(
        <code
          key={key++}
          className="bg-muted/30 px-1 py-0.5 rounded text-xs font-mono"
        >
          {part.slice(1, -1)}
        </code>
      );
    } else if (part) {
      // Simple bold (** __) and italic (* _)
      const boldParts = part.split(/(\*\*[^*]+\*\*|__[^_]+__)/g);
      for (const bp of boldParts) {
        if (bp.startsWith("**") && bp.endsWith("**")) {
          nodes.push(<strong key={key++}>{bp.slice(2, -2)}</strong>);
        } else if (bp.startsWith("__") && bp.endsWith("__")) {
          nodes.push(<strong key={key++}>{bp.slice(2, -2)}</strong>);
        } else if (bp) {
          const italicParts = bp.split(/(\*[^*]+\*|_[^_]+_)/g);
          for (const ip of italicParts) {
            if ((ip.startsWith("*") && ip.endsWith("*")) || (ip.startsWith("_") && ip.endsWith("_"))) {
              nodes.push(<em key={key++}>{ip.slice(1, -1)}</em>);
            } else if (ip) {
              nodes.push(<span key={key++}>{ip}</span>);
            }
          }
        }
      }
    }
  }

  return nodes;
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}
