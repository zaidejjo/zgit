import { type ReactNode } from "react";
import { cn } from "@/lib/utils";
import { AgentMessage, AgentToolCall } from "@/store/app";

interface ChatMessageProps {
  message: AgentMessage;
}

export default function ChatMessage({ message }: ChatMessageProps) {
  const isUser = message.role === "user";
  const isTool = message.role === "tool";
  const isAssistant = message.role === "assistant";

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
    <div className={cn("flex px-4 py-2", isUser ? "justify-end" : "justify-start")}>
      <div
        className={cn(
          "max-w-[85%] rounded-lg px-3.5 py-2 space-y-1.5",
          isUser
            ? "bg-primary text-primary-foreground rounded-br-sm"
            : "bg-muted/60 text-foreground rounded-bl-sm border border-border/40"
        )}
      >
        {message.tool_calls && message.tool_calls.length > 0 && (
          <div className="flex flex-wrap gap-1.5 mb-1.5">
            {message.tool_calls.map((tc) => (
              <ToolCallBadge key={tc.id} toolCall={tc} />
            ))}
          </div>
        )}

        {message.content ? (
          <MarkdownContent text={message.content} isUser={isUser} />
        ) : (
          message.tool_calls && message.tool_calls.length > 0 && (
            <p className="text-xs text-muted-foreground italic">
              Using tools...
            </p>
          )
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
          className="bg-black/20 dark:bg-black/40 rounded-md p-2.5 text-xs font-mono overflow-x-auto border border-border/30 my-1"
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
          className="bg-black/20 dark:bg-black/40 px-1 py-0.5 rounded text-xs font-mono"
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
