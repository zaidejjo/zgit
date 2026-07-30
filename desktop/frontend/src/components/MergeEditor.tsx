import { useState, useRef, useEffect, useCallback } from "react";
import { useAppStore, ConflictBlock } from "../store/app";
import {
  X, CheckCheck, ArrowRight, Save, AlertTriangle, Code2, FileText,
} from "lucide-react";

const LINE_HEIGHT = 22;

export default function MergeEditor() {
  const {
    mergeEditorOpen, mergeEditorFile, mergeConflictDetail,
    closeMergeEditor, resolveMergeBlock, saveResolvedFile, error,
  } = useAppStore();

  const [saving, setSaving] = useState(false);
  const [syncScroll, setSyncScroll] = useState(true);
  const [activeTab, setActiveTab] = useState<"editor" | "ours" | "theirs">("editor");

  const oursRef = useRef<HTMLDivElement>(null);
  const theirsRef = useRef<HTMLDivElement>(null);
  const resultRef = useRef<HTMLDivElement>(null);

  const handleScroll = useCallback((source: "ours" | "theirs" | "result") => {
    if (!syncScroll) return;
    const sources = { ours: oursRef, theirs: theirsRef, result: resultRef };
    const sourceEl = sources[source].current;
    if (!sourceEl) return;

    const scrollTop = sourceEl.scrollTop;
    Object.entries(sources).forEach(([key, ref]) => {
      if (key !== source && ref.current) {
        ref.current.scrollTop = scrollTop;
      }
    });
  }, [syncScroll]);

  if (!mergeEditorOpen || !mergeConflictDetail) return null;

  const detail = mergeConflictDetail;
  const allResolved = detail.blocks.every((b) => b.state !== "unresolved");

  const handleSave = async () => {
    setSaving(true);
    await saveResolvedFile();
    setSaving(false);
  };

  const renderBlockLineCount = (block: ConflictBlock) => {
    const ourLines = block.ours ? block.ours.split("\n").length : 0;
    const theirLines = block.theirs ? block.theirs.split("\n").length : 0;
    return { ourLines, theirLines };
  };

  return (
    <div className="fixed inset-0 z-50 bg-background/80 backdrop-blur-sm flex items-center justify-center p-2 sm:p-4">
      <div className="w-full max-w-6xl h-[90vh] bg-background border border-border rounded-xl shadow-2xl flex flex-col overflow-hidden">
        {/* Header */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-border bg-muted/30 shrink-0">
          <Code2 className="w-5 h-5 text-destructive" />
          <div className="flex-1 min-w-0">
            <h2 className="text-sm font-semibold truncate">Merge Conflict Editor</h2>
            <p className="text-xs text-muted-foreground font-mono truncate">{detail.path}</p>
          </div>

          {/* Sync scroll toggle */}
          <label className="flex items-center gap-1.5 text-xs text-muted-foreground cursor-pointer select-none">
            <input
              type="checkbox"
              checked={syncScroll}
              onChange={() => setSyncScroll(!syncScroll)}
              className="rounded border-border"
            />
            Sync scroll
          </label>

          {/* Tab switcher (mobile-friendly) */}
          <div className="flex items-center gap-0.5 bg-muted rounded-md p-0.5 text-xs">
            {(["ours", "theirs", "editor"] as const).map((tab) => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`px-2.5 py-1 rounded font-medium capitalize transition-colors ${
                  activeTab === tab ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {tab}
              </button>
            ))}
          </div>

          {/* Close */}
          <button
            onClick={closeMergeEditor}
            className="p-1.5 rounded-md hover:bg-muted transition-colors"
            title="Close"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Error banner */}
        {error && (
          <div className="flex items-center gap-2 px-4 py-2 bg-destructive/10 border-b border-destructive/20 text-xs text-destructive">
            <AlertTriangle className="w-3.5 h-3.5 shrink-0" />
            {error}
          </div>
        )}

        {/* 3-pane content */}
        <div className="flex-1 flex flex-col sm:flex-row overflow-hidden">
          {/* Left: Ours */}
          <div className="flex-1 flex flex-col min-w-0 border-r border-border">
            <div className="px-3 py-1.5 text-[11px] font-semibold text-muted-foreground uppercase tracking-wider bg-primary/5 border-b border-border shrink-0">
              <span className="text-primary">◀ Ours (Current)</span>
            </div>
            <div
              ref={oursRef}
              onScroll={() => handleScroll("ours")}
              className="flex-1 overflow-auto font-mono text-xs leading-[22px]"
            >
              <DiffLines
                content={detail.ours}
                blocks={detail.blocks}
                side="ours"
                highlightLines={detail.blocks.map((b) => ({
                  start: b.ours_start,
                  end: b.ours_end,
                  className: b.state === "use-ours" ? "bg-primary/20" : "bg-primary/10",
                }))}
              />
            </div>
          </div>

          {/* Right: Theirs */}
          <div className="flex-1 flex flex-col min-w-0 border-r border-border hidden sm:flex">
            <div className="px-3 py-1.5 text-[11px] font-semibold text-muted-foreground uppercase tracking-wider bg-[hsl(var(--pr-merged))/0.08] border-b border-border shrink-0">
              <span className="text-[hsl(var(--pr-merged))]">▶ Theirs (Incoming)</span>
            </div>
            <div
              ref={theirsRef}
              onScroll={() => handleScroll("theirs")}
              className="flex-1 overflow-auto font-mono text-xs leading-[22px]"
            >
              <DiffLines
                content={detail.theirs}
                blocks={detail.blocks}
                side="theirs"
                highlightLines={detail.blocks.map((b) => ({
                  start: b.theirs_start,
                  end: b.theirs_end,
                  className: b.state === "use-theirs" ? "bg-primary/20" : "bg-primary/10",
                }))}
              />
            </div>
          </div>

          {/* Bottom: Result (Resolved) */}
          <div className="flex-1 flex flex-col min-w-0">
            <div className="px-3 py-1.5 text-[11px] font-semibold text-muted-foreground uppercase tracking-wider bg-success/10 border-b border-border shrink-0">
              <span className="text-success">✔ Result</span>
              {!allResolved && (
                <span className="ml-2 text-warning font-normal">
                  ({detail.blocks.filter((b) => b.state === "unresolved").length} unresolved)
                </span>
              )}
            </div>
            <div
              ref={resultRef}
              onScroll={() => handleScroll("result")}
              className="flex-1 overflow-auto font-mono text-xs leading-[22px]"
            >
              {detail.blocks.length === 0 ? (
                <div className="p-4 text-muted-foreground text-xs italic">No conflicts detected in this file.</div>
              ) : (
                <ResultContent detail={detail} onResolve={resolveMergeBlock} />
              )}
            </div>
          </div>
        </div>

        {/* Bottom bar: block action buttons */}
        <div className="px-4 py-2.5 border-t border-border bg-muted/30 flex items-center gap-2 shrink-0 flex-wrap">
          <span className="text-xs text-muted-foreground mr-2">
            {detail.blocks.length} block{detail.blocks.length !== 1 ? "s" : ""}
            {allResolved ? " — all resolved" : ` — ${detail.blocks.filter((b) => b.state === "unresolved").length} remaining`}
          </span>

          <div className="flex-1" />

          {/* Use Ours All / Theirs All buttons */}
          {!allResolved && (
            <>
              <button
                onClick={() => {
                  detail.blocks.forEach((_, idx) => resolveMergeBlock(idx, "use-ours"));
                }}
                className="flex items-center gap-1 px-2.5 py-1.5 rounded text-xs font-medium bg-primary/10 text-primary hover:bg-primary/20 transition-colors"
              >
                <ArrowRight className="w-3 h-3" /> Use Ours All
              </button>
              <button
                onClick={() => {
                  detail.blocks.forEach((_, idx) => resolveMergeBlock(idx, "use-theirs"));
                }}
                className="flex items-center gap-1 px-2.5 py-1.5 rounded text-xs font-medium bg-[hsl(var(--pr-merged))/0.12] text-[hsl(var(--pr-merged))] hover:bg-[hsl(var(--pr-merged))/0.2] transition-colors"
              >
                Use Theirs All <ArrowRight className="w-3 h-3" />
              </button>
              <span className="text-muted-foreground text-xs">|</span>
            </>
          )}

          {/* Save & Stage */}
          <button
            onClick={handleSave}
            disabled={saving || !allResolved}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-semibold bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            {saving ? (
              <span className="w-3 h-3 border-2 border-current border-t-transparent rounded-full animate-spin" />
            ) : (
              <Save className="w-3.5 h-3.5" />
            )}
            {saving ? "Saving..." : allResolved ? "Save & Stage" : "Resolve all blocks"}
          </button>
        </div>
      </div>
    </div>
  );
}

/* ─── Result Content (interactive blocks) ─── */

function ResultContent({
  detail,
  onResolve,
}: {
  detail: { blocks: ConflictBlock[] };
  onResolve: (idx: number, state: "use-ours" | "use-theirs" | "edited", content?: string) => void;
}) {
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [editContent, setEditContent] = useState("");

  const handleEdit = (idx: number) => {
    if (editingIndex === idx) {
      onResolve(idx, "edited", editContent);
      setEditingIndex(null);
    } else {
      setEditingIndex(idx);
      setEditContent(detail.blocks[idx].resolved || detail.blocks[idx].ours);
    }
  };

  const lines: React.ReactNode[] = [];
  detail.blocks.forEach((block, idx) => {
    const resolved = block.resolved || block.ours;
    const resolvedLines = resolved ? resolved.split("\n") : [];

    // Add resolved lines
    resolvedLines.forEach((line, lineIdx) => {
      let bg = "";
      if (block.state === "use-ours") bg = "bg-blue-500/5";
      else if (block.state === "use-theirs") bg = "bg-purple-500/5";
      else if (block.state === "edited") bg = "bg-amber-500/5";
      else bg = "bg-destructive/10";

      // Insert block controls before the first line of the block
      if (lineIdx === 0) {
        lines.push(
          <div key={`block-${idx}-controls`} className={`px-2 py-1.5 ${bg} border-b border-t border-border/50`}>
            <BlockControls
              block={block}
              idx={idx}
              onResolve={onResolve}
              editing={editingIndex === idx}
              editContent={editContent}
              onEditContentChange={setEditContent}
              onEditToggle={() => handleEdit(idx)}
            />
          </div>
        );
      }

      lines.push(
        <div key={`block-${idx}-line-${lineIdx}`} className={`px-3 ${bg} hover:bg-foreground/5`}>
          <span className="select-none text-muted-foreground/30 mr-3 inline-block w-8 text-right">
            {lineIdx + 1}
          </span>
          {line || <span className="text-muted-foreground/30">⎵</span>}
        </div>
      );
    });
  });

  return <div className="py-2">{lines}</div>;
}

/* ─── Block Controls (per-conflict resolution toggles) ─── */

function BlockControls({
  block,
  idx,
  onResolve,
  editing,
  editContent,
  onEditContentChange,
  onEditToggle,
}: {
  block: ConflictBlock;
  idx: number;
  onResolve: (idx: number, state: "use-ours" | "use-theirs" | "edited", content?: string) => void;
  editing: boolean;
  editContent: string;
  onEditContentChange: (v: string) => void;
  onEditToggle: () => void;
}) {
  return (
    <div className="flex items-center gap-1.5 flex-wrap">
      <button
        onClick={() => onResolve(idx, "use-ours")}
        className={`text-[10px] px-2 py-0.5 rounded font-medium transition-colors ${
          block.state === "use-ours"
            ? "bg-blue-500/20 text-blue-600 dark:text-blue-400 ring-1 ring-blue-500/30"
            : "bg-blue-500/10 text-blue-600/60 hover:bg-blue-500/20"
        }`}
      >
        Ours
      </button>
      <button
        onClick={() => onResolve(idx, "use-theirs")}
        className={`text-[10px] px-2 py-0.5 rounded font-medium transition-colors ${
          block.state === "use-theirs"
            ? "bg-purple-500/20 text-purple-600 dark:text-purple-400 ring-1 ring-purple-500/30"
            : "bg-purple-500/10 text-purple-600/60 hover:bg-purple-500/20"
        }`}
      >
        Theirs
      </button>
      <button
        onClick={onEditToggle}
        className={`text-[10px] px-2 py-0.5 rounded font-medium transition-colors ${
          block.state === "edited"
            ? "bg-amber-500/20 text-amber-600 dark:text-amber-400 ring-1 ring-amber-500/30"
            : "bg-amber-500/10 text-amber-600/60 hover:bg-amber-500/20"
        }`}
      >
        {editing ? "Apply" : "Edit"}
      </button>

      {block.state === "unresolved" && (
        <span className="text-[10px] text-destructive/60 ml-1">⚠ unresolved</span>
      )}

      {editing && (
        <textarea
          value={editContent}
          onChange={(e) => onEditContentChange(e.target.value)}
          className="w-full mt-1 px-2 py-1 text-xs font-mono bg-background border border-border rounded resize-y min-h-[60px]"
          placeholder="Edit resolved content..."
          autoFocus
        />
      )}
    </div>
  );
}

/* ─── DiffLines (syntax-highlighted lines with block highlighting) ─── */

function DiffLines({
  content,
  blocks,
  side,
  highlightLines,
}: {
  content: string;
  blocks: ConflictBlock[];
  side: "ours" | "theirs";
  highlightLines: Array<{ start: number; end: number; className: string }>;
}) {
  const lines = content.split("\n");

  return (
    <div className="py-2">
      {lines.map((line, idx) => {
        const lineNum = idx + 1;
        // Check if this line is in a highlighted region
        const highlight = highlightLines.find(
          (h) => lineNum >= h.start && lineNum <= h.end
        );

        return (
          <div
            key={idx}
            className={`px-3 ${highlight?.className || ""} hover:bg-foreground/5`}
          >
            <span className="select-none text-muted-foreground/30 mr-3 inline-block w-8 text-right">
              {lineNum}
            </span>
            {line || <span className="text-muted-foreground/30">⎵</span>}
          </div>
        );
      })}
    </div>
  );
}
