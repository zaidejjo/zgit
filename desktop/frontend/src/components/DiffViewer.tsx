import { useState, useMemo } from "react";
import { FileText, Plus, Minus, ChevronDown, ChevronRight, Columns3, AlignJustify, ArrowUpFromLine } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/store/app";
import type { FileChange } from "@/store/app";

type DiffViewMode = "unified" | "split";

interface ParsedHunk {
  header: string;     // @@ -a,b +c,d @@
  oldStart: number;
  newStart: number;
  lines: HunkLine[];
  raw: string;        // full hunk text (header + body)
}

interface HunkLine {
  type: "context" | "add" | "del";
  content: string;      // the line without leading +/-/space
  oldLineNum: number | null;
  newLineNum: number | null;
}

interface DiffViewerProps {
  file: FileChange;
  defaultCollapsed?: boolean;
}

/** Parse unified-diff into hunks with line numbers */
function parseHunks(unifiedDiff: string): ParsedHunk[] {
  const lines = unifiedDiff.split("\n");
  const hunks: ParsedHunk[] = [];
  let current: ParsedHunk | null = null;
  let rawLines: string[] = [];

  for (const rawLine of lines) {
    if (rawLine.startsWith("@@")) {
      if (current) {
        current.raw = rawLines.join("\n");
        hunks.push(current);
      }
      rawLines = [rawLine];
      // @@ -oldStart,oldCount +newStart,newCount @@
      const m = rawLine.match(/@@\s+-(\d+)(?:,\d+)?\s+\+(\d+)(?:,\d+)?\s+@@/);
      current = {
        header: rawLine,
        oldStart: m ? parseInt(m[1], 10) : 1,
        newStart: m ? parseInt(m[2], 10) : 1,
        lines: [],
        raw: "",
      };
    } else if (current) {
      rawLines.push(rawLine);
      const ch = rawLine.charAt(0);
      const lineContent = rawLine.slice(1);
      if (ch === "+") {
        current.lines.push({
          type: "add",
          content: lineContent,
          oldLineNum: null,
          newLineNum: current.newStart + current.lines.filter((l) => l.type !== "del").length,
        });
      } else if (ch === "-") {
        current.lines.push({
          type: "del",
          content: lineContent,
          oldLineNum: current.oldStart + current.lines.filter((l) => l.type !== "add").length,
          newLineNum: null,
        });
      } else if (ch === " " || ch === "\\") {
        current.lines.push({
          type: "context",
          content: rawLine,
          oldLineNum: current.oldStart + current.lines.filter((l) => l.type !== "add").length,
          newLineNum: current.newStart + current.lines.filter((l) => l.type !== "del").length,
        });
      } else {
        // Fallback for anything else
        current.lines.push({
          type: "context",
          content: rawLine,
          oldLineNum: null,
          newLineNum: null,
        });
      }
    }
  }

  if (current) {
    current.raw = rawLines.join("\n");
    hunks.push(current);
  }

  return hunks;
}

/** Extract the diff header lines for a file to build a valid patch for git apply. */
function getFileDiffHeader(unifiedDiff: string): string {
  const lines = unifiedDiff.split("\n");
  const headerLines: string[] = [];
  for (const line of lines) {
    if (line.startsWith("@@")) break;
    headerLines.push(line);
  }
  return headerLines.join("\n");
}

function DiffViewer({ file, defaultCollapsed = false }: DiffViewerProps) {
  const [viewMode, setViewMode] = useState<DiffViewMode>("unified");
  const [collapsed, setCollapsed] = useState(defaultCollapsed);
  const stagePatch = useAppStore((s) => s.stagePatch);

  const hunks = useMemo(() => {
    if (!file.unified_diff) return [];
    return parseHunks(file.unified_diff);
  }, [file.unified_diff]);

  const diffHeader = useMemo(() => {
    if (!file.unified_diff) return "";
    return getFileDiffHeader(file.unified_diff);
  }, [file.unified_diff]);

  if (!file.unified_diff) {
    return (
      <div className="py-4 text-muted-foreground italic text-xs">
        No diff content available
      </div>
    );
  }

  const handleStageHunk = async (hunkRaw: string) => {
    // Build a standalone patch: file header + hunk
    const patch = `${diffHeader}\n${hunkRaw}\n`;
    await stagePatch(patch);
  };

  const fileName = file.new_path || file.old_path || "";

  return (
    <div className="mb-2 rounded-lg overflow-hidden border border-border/30 bg-card/30">
      {/* File header */}
      <div
        className="flex items-center gap-2 px-3 py-2 bg-muted/10 border-b border-border/20 cursor-pointer hover:bg-muted/20 transition-all duration-150 select-none"
        onClick={() => setCollapsed(!collapsed)}
      >
        <button className="shrink-0 text-muted-foreground hover:text-foreground transition-colors">
          {collapsed ? <ChevronRight className="w-3.5 h-3.5" /> : <ChevronDown className="w-3.5 h-3.5" />}
        </button>
        <FileText className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
        <span className="text-xs font-mono flex-1 truncate text-foreground/80">{fileName}</span>
        <span className="text-[11px] text-success font-medium">+{file.additions}</span>
        <span className="text-[11px] text-destructive font-medium">-{file.deletions}</span>
        {/* View mode toggle */}
        <div className="flex gap-0.5 ml-2" onClick={(e) => e.stopPropagation()}>
          <button
            className={cn(
              "p-1 rounded transition-colors",
              viewMode === "unified"
                ? "bg-accent/60 text-accent-foreground"
                : "text-muted-foreground hover:text-foreground hover:bg-accent/30"
            )}
            onClick={() => setViewMode("unified")}
            title="Unified view"
          >
            <AlignJustify className="w-3.5 h-3.5" />
          </button>
          <button
            className={cn(
              "p-1 rounded transition-colors",
              viewMode === "split"
                ? "bg-accent/60 text-accent-foreground"
                : "text-muted-foreground hover:text-foreground hover:bg-accent/30"
            )}
            onClick={() => setViewMode("split")}
            title="Split view"
          >
            <Columns3 className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {/* Diff body */}
      {!collapsed && (
        <div className="text-xs font-mono leading-relaxed">
          {viewMode === "unified" ? (
            <UnifiedView hunks={hunks} onStageHunk={handleStageHunk} />
          ) : (
            <SplitView hunks={hunks} onStageHunk={handleStageHunk} />
          )}
        </div>
      )}
    </div>
  );
}

/* ─── Unified View ─── */

function UnifiedView({
  hunks,
  onStageHunk,
}: {
  hunks: ParsedHunk[];
  onStageHunk: (raw: string) => void;
}) {
  return (
    <div className="divide-y divide-border/20">
      {hunks.map((hunk, hunkIdx) => (
        <div key={hunkIdx} className="relative group">
          {/* Hunk header */}
          <div className="flex items-center gap-2 px-3 py-1 bg-blue-500/[0.04] text-blue-500 text-[10px] font-semibold border-b border-blue-500/10">
            <span>{hunk.header}</span>
            <div className="flex-1" />
            <button
              className="opacity-0 group-hover:opacity-100 transition-all duration-150 flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] text-success hover:bg-success/10 border border-transparent hover:border-success/30 press-scale"
              onClick={() => onStageHunk(hunk.raw)}
              title="Stage this hunk"
            >
              <ArrowUpFromLine className="w-3 h-3" />
              Stage hunk
            </button>
          </div>
          {hunk.lines.map((line, lineIdx) => (
            <UnifiedLine key={lineIdx} line={line} />
          ))}
        </div>
      ))}
    </div>
  );
}

function UnifiedLine({ line }: { line: HunkLine }) {
  const isAdd = line.type === "add";
  const isDel = line.type === "del";
  const lineNum = isAdd ? line.newLineNum : line.oldLineNum;

  return (
    <div
      className={cn(
        "flex items-stretch min-h-[1.375rem]",
        isAdd && "bg-success/[0.06] border-l-2 border-l-success/60",
        isDel && "bg-destructive/[0.06] border-l-2 border-l-destructive/60",
        line.type === "context" && "border-l-2 border-l-transparent"
      )}
    >
      {/* Line number */}
      <span className="w-12 shrink-0 text-right pr-2 text-[10px] text-muted-foreground/50 select-none leading-[1.375rem] bg-muted/10">
        {lineNum ?? ""}
      </span>
      {/* Sign */}
      <span
        className={cn(
          "w-4 shrink-0 text-center select-none leading-[1.375rem] text-[10px]",
          isAdd && "text-success",
          isDel && "text-destructive"
        )}
      >
        {isAdd ? "+" : isDel ? "-" : " "}
      </span>
      {/* Content with syntax highlighting */}
      <span className="flex-1 pl-2 whitespace-pre leading-[1.375rem] text-[11px]">
        <SyntaxHighlight code={line.content} />
      </span>
    </div>
  );
}

/* ─── Split View ─── */

function SplitView({
  hunks,
  onStageHunk,
}: {
  hunks: ParsedHunk[];
  onStageHunk: (raw: string) => void;
}) {
  return (
    <div className="divide-y divide-border/20">
      {hunks.map((hunk, hunkIdx) => {
        // Build left (old) and right (new) columns
        const leftLines: Array<{ type: "context" | "del"; content: string; lineNum: number | null }> = [];
        const rightLines: Array<{ type: "context" | "add"; content: string; lineNum: number | null }> = [];
        let oldNum = hunk.oldStart;
        let newNum = hunk.newStart;

        for (const line of hunk.lines) {
          if (line.type === "context") {
            leftLines.push({ type: "context", content: line.content, lineNum: oldNum });
            rightLines.push({ type: "context", content: line.content, lineNum: newNum });
            oldNum++;
            newNum++;
          } else if (line.type === "del") {
            leftLines.push({ type: "del", content: line.content, lineNum: oldNum });
            rightLines.push({ type: "context", content: "", lineNum: null });
            oldNum++;
          } else if (line.type === "add") {
            leftLines.push({ type: "context", content: "", lineNum: null });
            rightLines.push({ type: "add", content: line.content, lineNum: newNum });
            newNum++;
          }
        }

        return (
          <div key={hunkIdx} className="relative group">
            {/* Hunk header with stage button */}
            <div className="flex items-center gap-2 px-3 py-1 bg-blue-500/[0.04] text-blue-500 text-[10px] font-semibold border-b border-blue-500/10">
              <span>{hunk.header}</span>
              <div className="flex-1" />
              <button
                className="opacity-0 group-hover:opacity-100 transition-all duration-150 flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] text-success hover:bg-success/10 border border-transparent hover:border-success/30 press-scale"
                onClick={() => onStageHunk(hunk.raw)}
                title="Stage this hunk"
              >
                <ArrowUpFromLine className="w-3 h-3" />
                Stage hunk
              </button>
            </div>
            {/* Side-by-side columns */}
            <div className="flex">
              {/* Left column (old) */}
              <div className="w-1/2 border-r border-border/20">
                {leftLines.map((line, i) => (
                  <SplitLine key={i} line={line} side="left" />
                ))}
              </div>
              {/* Right column (new) */}
              <div className="w-1/2">
                {rightLines.map((line, i) => (
                  <SplitLine key={i} line={line} side="right" />
                ))}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}

function SplitLine({
  line,
  side,
}: {
  line: { type: "context" | "del" | "add"; content: string; lineNum: number | null };
  side: "left" | "right";
}) {
  const isHighlighted = line.type === "del" || line.type === "add";

  return (
    <div
      className={cn(
        "flex items-stretch min-h-[1.375rem]",
        line.type === "del" && "bg-destructive/[0.06]",
        line.type === "add" && "bg-success/[0.06]"
      )}
    >
      {/* Line number */}
      <span className="w-10 shrink-0 text-right pr-1 text-[10px] text-muted-foreground/50 select-none leading-[1.375rem] bg-muted/10">
        {line.lineNum ?? ""}
      </span>
      {/* Sign */}
      {line.content !== "" ? (
        <span
          className={cn(
            "w-3 shrink-0 text-center select-none leading-[1.375rem] text-[10px]",
            line.type === "del" && "text-destructive",
            line.type === "add" && "text-success"
          )}
        >
          {line.type === "del" ? "-" : line.type === "add" ? "+" : " "}
        </span>
      ) : (
        <span className="w-3 shrink-0" />
      )}
      {/* Content */}
      <span className="flex-1 pl-1.5 whitespace-pre leading-[1.375rem] text-[11px]">
        {isHighlighted ? <SyntaxHighlight code={line.content} /> : line.content}
      </span>
    </div>
  );
}

/* ─── Syntax Highlighting (CSS-based) ─── */

const SYNTAX_PATTERNS = [
  // Keywords
  { pattern: /\b(function|const|let|var|if|else|for|while|return|import|export|from|class|extends|new|this|async|await|try|catch|throw|type|interface|enum|package|switch|case|default|break|continue|def|end|do|then|when|module|where|nil|true|false|null|undefined)\b/g, className: "text-purple-600 dark:text-purple-400" },
  // Types
  { pattern: /\b(string|number|boolean|int|float|void|any|never|Record|Partial|Pick|Omit|Required|Readonly)\b/g, className: "text-blue-600 dark:text-blue-400" },
  // Strings
  { pattern: /("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|`(?:[^`\\]|\\.)*`)/g, className: "text-green-600 dark:text-green-400" },
  // Numbers
  { pattern: /\b(\d+\.?\d*)\b/g, className: "text-orange-500 dark:text-orange-400" },
  // Comments
  { pattern: /(\/\/.*$|\/\*[\s\S]*?\*\/)/g, className: "text-muted-foreground italic" },
  // Decorators / annotations
  { pattern: /(@\w+)/g, className: "text-yellow-600 dark:text-yellow-400" },
];

function SyntaxHighlight({ code }: { code: string }) {
  // For empty code in split view, render nothing
  if (!code) return null;

  const parts: Array<{ text: string; className?: string }> = [{ text: code }];

  for (const { pattern, className } of SYNTAX_PATTERNS) {
    let i = 0;
    while (i < parts.length) {
      const part = parts[i];
      if (part.className) {
        i++;
        continue;
      }
      const matches = [...part.text.matchAll(pattern)];
      if (matches.length === 0) {
        i++;
        continue;
      }

      const newParts: Array<{ text: string; className?: string }> = [];
      let lastIdx = 0;
      for (const match of matches) {
        if (match.index! > lastIdx) {
          newParts.push({ text: part.text.slice(lastIdx, match.index) });
        }
        const matchedText = match[1] || match[0];
        newParts.push({ text: matchedText, className });
        lastIdx = match.index! + match[0].length;
      }
      if (lastIdx < part.text.length) {
        newParts.push({ text: part.text.slice(lastIdx) });
      }

      parts.splice(i, 1, ...newParts);
      i += newParts.length;
    }
  }

  return (
    <>
      {parts.map((part, i) =>
        part.className ? (
          <span key={i} className={part.className}>{part.text}</span>
        ) : (
          <span key={i}>{part.text}</span>
        )
      )}
    </>
  );
}

export default DiffViewer;
