import { useEffect, useState, useCallback } from "react";
import {
  Download, Upload, Undo2, Archive, RotateCcw, Play, X,
  GitCommitHorizontal, SquarePen, AlignLeft, CheckCheck,
  AlertTriangle, ChevronDown, ListChecks, FileText,
} from "lucide-react";
import { useAppStore } from "@/store/app";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Checkbox } from "@/components/ui/checkbox";
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from "@/components/ui/tooltip";
import {
  DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

const CONVENTIONAL_PREFIXES = [
  { value: "feat", label: "feat", desc: "New feature", color: "text-green-500" },
  { value: "fix", label: "fix", desc: "Bug fix", color: "text-red-500" },
  { value: "docs", label: "docs", desc: "Documentation", color: "text-blue-500" },
  { value: "style", label: "style", desc: "Formatting", color: "text-purple-500" },
  { value: "refactor", label: "refactor", desc: "Code restructure", color: "text-yellow-500" },
  { value: "test", label: "test", desc: "Tests", color: "text-orange-500" },
  { value: "chore", label: "chore", desc: "Maintenance", color: "text-muted-foreground" },
] as const;

// StatusType enum values from Go models
const STATUS_UNTRACKED = 0;
const STATUS_UNMODIFIED = 7;

export default function StatusPage() {
  const {
    status, diff, loading, fetchStatus, fetchDiff,
    stageFile, unstageFile, stageAll, unstageAll, clearDiff,
    commitAndPush, commit, discardFile,
    stashes, fetchStashes, stashPush, stashPop, stashApply, stashDrop,
    gitFetch, gitPull, gitPushForce,
  } = useAppStore();

  // Diff selection
  const [selectedFile, setSelectedFile] = useState<string | null>(null);

  // Commit form state
  const [summary, setSummary] = useState("");
  const [description, setDescription] = useState("");
  const [committing, setCommitting] = useState(false);

  // Stash state
  const [stashMsg, setStashMsg] = useState("");
  const [stashing, setStashing] = useState(false);

  // Multi-select staging: set of file paths checked
  const [checkedFiles, setCheckedFiles] = useState<Set<string>>(new Set());

  useEffect(() => {
    fetchStatus();
    fetchStashes();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Reset checkbox selections when status refreshes
  useEffect(() => {
    setCheckedFiles(new Set());
  }, [status?.files]);

  const handleFileClick = (path: string) => {
    setSelectedFile(path);
    fetchDiff(path);
  };

  const handleCloseDiff = () => {
    setSelectedFile(null);
    clearDiff();
  };

  const toggleCheck = (path: string) => {
    setCheckedFiles((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  };

  const handleStageChecked = useCallback(async () => {
    const files = Array.from(checkedFiles);
    if (files.length === 0) return;
    // Stage all checked files in sequence
    for (const f of files) {
      await stageFile(f);
    }
  }, [checkedFiles, stageFile]);

  const handleUnstageChecked = useCallback(async () => {
    const files = Array.from(checkedFiles);
    if (files.length === 0) return;
    for (const f of files) {
      await unstageFile(f);
    }
  }, [checkedFiles, unstageFile]);

  const handleCommitClick = async (withPush: boolean) => {
    if (!summary.trim()) return;
    setCommitting(true);
    const fn = withPush ? commitAndPush : commit;
    await fn(summary.trim(), description);
    setCommitting(false);
    setSummary("");
    setDescription("");
  };

  const handleForcePushThenPush = async () => {
    // Force push first, then do a normal commit+push sequence if needed
    // This is for when user wants to push after force push
    try {
      await gitPushForce();
    } catch {
      // Error handled in store
    }
  };

  const handleStashPush = async () => {
    setStashing(true);
    await stashPush(stashMsg);
    setStashing(false);
    setStashMsg("");
  };

  if (!status) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        {loading.status ? "Loading status..." : "No repository open"}
      </div>
    );
  }

  const stagedFiles = (status.files || []).filter(
    (f) => f.staged !== STATUS_UNMODIFIED && f.staged !== STATUS_UNTRACKED
  );
  const unstagedFiles = (status.files || []).filter(
    (f) => f.unstaged !== STATUS_UNMODIFIED
  );
  const untrackedFiles = (status.files || []).filter(
    (f) => f.staged === STATUS_UNTRACKED
  );

  const hasChecked = checkedFiles.size > 0;
  const checkedStaged = Array.from(checkedFiles).filter((p) =>
    stagedFiles.some((f) => f.path === p)
  );
  const checkedUnstaged = Array.from(checkedFiles).filter((p) =>
    unstagedFiles.some((f) => f.path === p) || untrackedFiles.some((f) => f.path === p)
  );

  const canCommit = stagedFiles.length > 0 && summary.trim().length > 0;

  return (
    <TooltipProvider delayDuration={300}>
      <div className="flex gap-4 h-full">
        {/* Left column: file list */}
        <div className="flex-1 min-w-0 flex flex-col">
          {/* Header row */}
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <h2 className="text-xl font-bold">Status</h2>
              {status.branch && (
                <Badge variant="outline" className="font-mono">
                  {status.branch}
                </Badge>
              )}
              {!status.is_clean && (
                <Badge variant="warning">
                  {status.ahead > 0 ? `+${status.ahead}` : ""}
                  {status.behind > 0 ? ` -${status.behind}` : ""}
                </Badge>
              )}
            </div>
            <div className="flex gap-1 items-center">
              <Button variant="ghost" size="sm" className="h-7 px-2 text-xs"
                onClick={() => gitFetch()} disabled={loading.fetch}>
                <Download className="w-3.5 h-3.5 mr-1" />
                Fetch
              </Button>
              <Button variant="ghost" size="sm" className="h-7 px-2 text-xs"
                onClick={() => gitPull(false)} disabled={loading.pull}>
                <Upload className="w-3.5 h-3.5 mr-1" />
                Pull
              </Button>
              <div className="w-px h-5 bg-border mx-1" />
              {/* Batch stage/unstage buttons */}
              {hasChecked ? (
                <>
                  {checkedStaged.length > 0 && (
                    <Button variant="outline" size="sm" onClick={handleUnstageChecked}>
                      Unstage Selected ({checkedStaged.length})
                    </Button>
                  )}
                  {checkedUnstaged.length > 0 && (
                    <Button variant="outline" size="sm" onClick={handleStageChecked}>
                      Stage Selected ({checkedUnstaged.length})
                    </Button>
                  )}
                </>
              ) : (
                <>
                  <Button variant="outline" size="sm" onClick={() => stageAll()}>
                    Stage All
                  </Button>
                  <Button variant="outline" size="sm" onClick={() => unstageAll()}>
                    Unstage All
                  </Button>
                </>
              )}
            </div>
          </div>

          {/* File list scroll area */}
          <ScrollArea className="flex-1 min-h-0">
            {status.is_clean && (
              <Card>
                <CardContent className="py-8 text-center text-muted-foreground">
                  Clean working tree
                </CardContent>
              </Card>
            )}

            {!status.is_clean && (
              <>
                {/* Staged files */}
                {stagedFiles.length > 0 && (
                  <div className="mb-4">
                    <h3 className="text-sm font-medium text-green-600 dark:text-green-400 mb-2">
                      Staged ({stagedFiles.length})
                    </h3>
                    {stagedFiles.map((f) => (
                      <FileRow
                        key={f.path}
                        file={f}
                        type="staged"
                        selected={selectedFile === f.path}
                        checked={checkedFiles.has(f.path)}
                        onToggleCheck={() => toggleCheck(f.path)}
                        onClick={() => handleFileClick(f.path)}
                        onStage={() => unstageFile(f.path)}
                      />
                    ))}
                  </div>
                )}

                {/* Staged counter */}
                {stagedFiles.length > 0 && (
                  <div className="flex items-center gap-1.5 mb-4 px-3 py-2 rounded-lg bg-primary/5 border border-primary/10 text-sm text-primary">
                    <ListChecks className="w-4 h-4" />
                    <span>
                      <strong>{stagedFiles.length}</strong> file{stagedFiles.length !== 1 ? "s" : ""} staged for commit
                    </span>
                  </div>
                )}

                {/* Unstaged files */}
                {unstagedFiles.length > 0 && (
                  <div className="mb-4">
                    <h3 className="text-sm font-medium text-yellow-600 dark:text-yellow-400 mb-2">
                      Unstaged ({unstagedFiles.length})
                    </h3>
                    {unstagedFiles.map((f) => (
                      <FileRow
                        key={f.path}
                        file={f}
                        type="unstaged"
                        selected={selectedFile === f.path}
                        checked={checkedFiles.has(f.path)}
                        onToggleCheck={() => toggleCheck(f.path)}
                        onClick={() => handleFileClick(f.path)}
                        onStage={() => stageFile(f.path)}
                        onDiscard={() => discardFile(f.path)}
                      />
                    ))}
                  </div>
                )}

                {/* Untracked files */}
                {untrackedFiles.length > 0 && (
                  <div className="mb-4">
                    <h3 className="text-sm font-medium text-muted-foreground mb-2">
                      Untracked ({untrackedFiles.length})
                    </h3>
                    {untrackedFiles.map((f) => (
                      <FileRow
                        key={f.path}
                        file={f}
                        type="untracked"
                        selected={selectedFile === f.path}
                        checked={checkedFiles.has(f.path)}
                        onToggleCheck={() => toggleCheck(f.path)}
                        onClick={() => handleFileClick(f.path)}
                        onStage={() => stageFile(f.path)}
                        onDiscard={() => discardFile(f.path)}
                      />
                    ))}
                  </div>
                )}
              </>
            )}
          </ScrollArea>
        </div>

        {/* Right column: commit + stash + diff */}
        <div className="w-[30rem] min-w-0 flex flex-col gap-4">
          {/* Commit section — always visible */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm flex items-center gap-2">
                <GitCommitHorizontal className="w-4 h-4" />
                Commit to{" "}
                <span className="font-mono text-primary">{status.branch}</span>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {/* Conventional commit chips */}
              <ConventionalChips
                summary={summary}
                onApplyPrefix={(prefix) => {
                  // Apply or remove prefix from summary
                  const regex = /^(feat|fix|docs|style|refactor|test|chore)(\([^)]*\))?:\s*/;
                  if (regex.test(summary)) {
                    // Replace existing prefix
                    setSummary(summary.replace(regex, `${prefix}: `));
                  } else {
                    // Prepend prefix
                    setSummary(`${prefix}: ${summary}`);
                  }
                }}
                onRemovePrefix={() => {
                  setSummary(summary.replace(/^(feat|fix|docs|style|refactor|test|chore)(\([^)]*\))?:\s*/, ""));
                }}
              />

              {/* Summary input */}
              <div className="relative">
                <SquarePen className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
                <input
                  className="flex h-10 w-full rounded-md border border-input bg-background pl-9 pr-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  placeholder="Summary (required)"
                  value={summary}
                  onChange={(e) => setSummary(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && !e.shiftKey) {
                      e.preventDefault();
                      if (canCommit && !committing) handleCommitClick(false);
                    }
                  }}
                  autoFocus
                />
              </div>

              {/* Description textarea */}
              <div className="relative">
                <AlignLeft className="absolute left-3 top-3 w-4 h-4 text-muted-foreground pointer-events-none" />
                <textarea
                  className="flex w-full rounded-md border border-input bg-background pl-9 pr-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring resize-none"
                  placeholder="Description (optional)"
                  rows={3}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                />
              </div>

              {/* Commit action buttons */}
              <div className="flex gap-2">
                {/* Primary: Commit to branch */}
                <Tooltip open={!canCommit ? undefined : false}>
                  <TooltipTrigger asChild>
                    <span tabIndex={0} className="flex-1">
                      <Button
                        size="sm"
                        className="w-full"
                        disabled={!canCommit || committing}
                        onClick={() => handleCommitClick(false)}
                      >
                        <CheckCheck className="w-4 h-4 mr-1.5" />
                        {committing ? "Committing..." : `Commit to ${status.branch}`}
                      </Button>
                    </span>
                  </TooltipTrigger>
                  {!canCommit && (
                    <TooltipContent side="bottom">
                      {stagedFiles.length === 0
                        ? "Stage at least one file first"
                        : "Enter a summary message"}
                    </TooltipContent>
                  )}
                </Tooltip>

                {/* Secondary: Commit & Push with force push dropdown */}
                <div className="flex">
                  <Button
                    variant="secondary"
                    size="sm"
                    className="rounded-r-none"
                    disabled={!canCommit || committing}
                    onClick={() => handleCommitClick(true)}
                  >
                    <Upload className="w-4 h-4 mr-1.5" />
                    {committing ? "Committing..." : "Commit & Push"}
                  </Button>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        variant="secondary"
                        size="sm"
                        className="rounded-l-none border-l border-secondary-foreground/20 px-2"
                        disabled={committing}
                        aria-label="Push options"
                      >
                        <ChevronDown className="w-3.5 h-3.5" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" side="bottom">
                      <DropdownMenuItem
                        disabled={!canCommit || committing}
                        onClick={() => handleCommitClick(true)}
                      >
                        <Upload className="w-4 h-4 mr-2" />
                        Commit & Push
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        onClick={handleForcePushThenPush}
                        className="text-destructive focus:text-destructive"
                      >
                        <AlertTriangle className="w-4 h-4 mr-2" />
                        Force Push
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Stash section */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm flex items-center gap-2">
                <Archive className="w-4 h-4" />
                Stash
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              {!status.is_clean && (
                <div className="flex gap-2">
                  <input
                    className="flex-1 rounded border border-input bg-background px-2 py-1.5 text-xs"
                    placeholder="Stash message (optional)"
                    value={stashMsg}
                    onChange={(e) => setStashMsg(e.target.value)}
                    onKeyDown={(e) => { if (e.key === "Enter") handleStashPush(); }}
                  />
                  <Button size="sm" className="h-7 text-xs" onClick={handleStashPush} disabled={stashing}>
                    <RotateCcw className="w-3 h-3 mr-1" />
                    {stashing ? "..." : "Stash"}
                  </Button>
                </div>
              )}

              {stashes.length === 0 ? (
                <p className="text-xs text-muted-foreground">No stashes</p>
              ) : (
                <ScrollArea className="max-h-48">
                  <div className="space-y-1">
                    {stashes.map((s) => (
                      <div key={s.index}
                        className="flex items-center gap-2 px-2 py-1.5 rounded text-xs hover:bg-accent/50 group">
                        <span className="font-mono text-muted-foreground shrink-0">stash@{s.index}</span>
                        <span className="flex-1 truncate">{s.message || "(no message)"}</span>
                        <div className="flex gap-0.5 opacity-0 group-hover:opacity-100">
                          <button className="h-5 px-1 text-muted-foreground hover:text-foreground"
                            onClick={() => stashPop(s.index)} title="Pop">
                            <Upload className="w-3 h-3" />
                          </button>
                          <button className="h-5 px-1 text-muted-foreground hover:text-foreground"
                            onClick={() => stashApply(s.index)} title="Apply">
                            <Play className="w-3 h-3" />
                          </button>
                          <button className="h-5 px-1 text-muted-foreground hover:text-destructive"
                            onClick={() => stashDrop(s.index)} title="Drop">
                            <Archive className="w-3 h-3" />
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </ScrollArea>
              )}
            </CardContent>
          </Card>

          {/* Diff panel */}
          {selectedFile && diff && (
            <div className="flex-1 min-h-0 border rounded-lg p-3">
              <div className="flex items-center justify-between mb-2">
                <h3 className="text-sm font-semibold font-mono truncate flex items-center gap-1.5">
                  <FileText className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                  {selectedFile}
                </h3>
                <div className="flex items-center gap-2 shrink-0">
                  <span className="text-xs text-green-600">+{diff.total_additions}</span>
                  <span className="text-xs text-red-600">-{diff.total_deletions}</span>
                  <Button variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={handleCloseDiff}>
                    <X className="w-3.5 h-3.5" />
                  </Button>
                </div>
              </div>
              <ScrollArea className="h-[calc(100vh-28rem)]">
                <pre className="text-xs font-mono leading-relaxed">
                  {(diff.files || []).map((f, idx) => (
                    <DiffBlock key={idx} file={f} />
                  ))}
                </pre>
              </ScrollArea>
            </div>
          )}
        </div>
      </div>
    </TooltipProvider>
  );
}

/* ─── Conventional Commit Chips ─── */

function ConventionalChips({
  summary,
  onApplyPrefix,
  onRemovePrefix,
}: {
  summary: string;
  onApplyPrefix: (prefix: string) => void;
  onRemovePrefix: () => void;
}) {
  const currentPrefix = CONVENTIONAL_PREFIXES.find((p) =>
    summary.startsWith(`${p.value}:`) || summary.startsWith(`${p.value}(`)
  );

  return (
    <div className="flex flex-wrap gap-1">
      {CONVENTIONAL_PREFIXES.map((p) => {
        const isActive = currentPrefix?.value === p.value;
        return (
          <button
            key={p.value}
            className={cn(
              "text-xs px-1.5 py-0.5 rounded border transition-colors",
              isActive
                ? "bg-primary/10 border-primary text-primary"
                : "border-border text-muted-foreground hover:border-primary/50 hover:text-foreground"
            )}
            onClick={() => {
              if (isActive) {
                onRemovePrefix();
              } else {
                onApplyPrefix(p.value);
              }
            }}
            title={p.desc}
          >
            <span className={p.color}>{p.value}</span>
          </button>
        );
      })}
      {currentPrefix && (
        <button
          className="text-xs px-1.5 py-0.5 rounded border border-destructive/50 text-destructive hover:bg-destructive/10 transition-colors"
          onClick={onRemovePrefix}
          title="Remove prefix"
        >
          <X className="w-3 h-3" />
        </button>
      )}
    </div>
  );
}

/* ─── File Row ─── */

interface FileRowProps {
  file: { path: string; staged: number; unstaged: number };
  type: "staged" | "unstaged" | "untracked";
  selected: boolean;
  checked: boolean;
  onToggleCheck: () => void;
  onClick: () => void;
  onStage: () => void;
  onDiscard?: () => void;
}

function FileRow({ file, type, selected, checked, onToggleCheck, onClick, onStage, onDiscard }: FileRowProps) {
  const colorClass = {
    staged: "border-l-green-500",
    unstaged: "border-l-yellow-500",
    untracked: "border-l-muted-foreground",
  }[type];

  const label = {
    staged: "M",
    unstaged: "M",
    untracked: "?",
  }[type];

  const labelColor = {
    staged: "text-green-500",
    unstaged: "text-yellow-500",
    untracked: "text-muted-foreground",
  }[type];

  return (
    <div
      className={cn(
        "group flex items-center gap-2 px-3 py-1.5 border-l-2 cursor-pointer hover:bg-accent/50 text-sm transition-colors",
        colorClass,
        selected && "bg-accent"
      )}
    >
      {/* Checkbox */}
      <div className="flex items-center shrink-0" onClick={(e) => e.stopPropagation()}>
        <Checkbox
          checked={checked}
          onChange={onToggleCheck}
          className="cursor-pointer"
        />
      </div>

      {/* Status label */}
      <span className={cn("w-5 text-center font-mono text-xs", labelColor)}>
        {label}
      </span>

      {/* File path — click to view diff */}
      <span className="flex-1 truncate" onClick={onClick}>
        {file.path}
      </span>

      {/* Actions */}
      <div className="flex gap-0.5 opacity-0 group-hover:opacity-100" onClick={(e) => e.stopPropagation()}>
        {type !== "staged" && onDiscard && (
          <button
            className="h-6 px-1.5 text-xs text-muted-foreground hover:text-destructive transition-colors"
            onClick={onDiscard}
            title="Discard changes"
          >
            <Undo2 className="w-3 h-3" />
          </button>
        )}
        <Button
          variant="ghost"
          size="sm"
          className="h-6 px-2 text-xs"
          onClick={onStage}
        >
          {type === "staged" ? "Unstage" : "Stage"}
        </Button>
      </div>
    </div>
  );
}

/* ─── Diff Block ─── */

function DiffBlock({ file }: { file: { new_path?: string; unified_diff?: string; additions: number; deletions: number } }) {
  if (!file.unified_diff) {
    return (
      <div className="py-4 text-muted-foreground italic">
        No diff content available
      </div>
    );
  }

  return (
    <div className="mb-4">
      <div className="flex gap-2 text-xs mb-1">
        <span className="text-green-500">+{file.additions}</span>
        <span className="text-red-500">-{file.deletions}</span>
        <span className="text-muted-foreground">{file.new_path || ""}</span>
      </div>
      <div className="bg-muted/30 rounded p-2 overflow-x-auto">
        <code className="text-xs">
          {file.unified_diff.split("\n").map((line, i) => {
            let lineClass = "";
            if (line.startsWith("+") && !line.startsWith("+++")) lineClass = "text-green-500 bg-green-500/10";
            else if (line.startsWith("-") && !line.startsWith("---")) lineClass = "text-red-500 bg-red-500/10";
            else if (line.startsWith("@")) lineClass = "text-blue-500";
            else if (line.startsWith("diff") || line.startsWith("---") || line.startsWith("+++")) lineClass = "text-muted-foreground";

            return (
              <div key={i} className={cn("whitespace-pre", lineClass)}>
                {line}
              </div>
            );
          })}
        </code>
      </div>
    </div>
  );
}
