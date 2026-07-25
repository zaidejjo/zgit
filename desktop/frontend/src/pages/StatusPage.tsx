import { useEffect, useState } from "react";
import { useAppStore } from "@/store/app";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";

export default function StatusPage() {
  const { status, diff, loading, fetchStatus, fetchDiff, stageFile, unstageFile, stageAll, unstageAll, clearDiff } =
    useAppStore();
  const [selectedFile, setSelectedFile] = useState<string | null>(null);

  useEffect(() => {
    fetchStatus();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleFileClick = (path: string) => {
    setSelectedFile(path);
    fetchDiff(path);
  };

  const handleCloseDiff = () => {
    setSelectedFile(null);
    clearDiff();
  };

  if (!status) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        {loading.status ? "Loading status..." : "No repository open"}
      </div>
    );
  }

  // StatusType enum values from Go models:
  // 0=Untracked, 1=Added, 2=Modified, 3=Deleted, 4=Renamed,
  // 5=Copied, 6=UpdatedButUnmerged, 7=Unmodified, 8=Ignored
  const STATUS_UNTRACKED = 0;
  const STATUS_UNMODIFIED = 7;
  const STATUS_IGNORED = 8;

  const stagedFiles = (status.files || []).filter(
    (f) => f.staged !== STATUS_UNMODIFIED && f.staged !== STATUS_UNTRACKED
  );
  const unstagedFiles = (status.files || []).filter(
    (f) => f.unstaged !== STATUS_UNMODIFIED
  );
  const untrackedFiles = (status.files || []).filter(
    (f) => f.staged === STATUS_UNTRACKED
  );

  return (
    <div className="flex gap-4 h-full">
      {/* File list */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <h2 className="text-xl font-bold">
              Status
            </h2>
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
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={() => stageAll()}>
              Stage All
            </Button>
            <Button variant="outline" size="sm" onClick={() => unstageAll()}>
              Unstage All
            </Button>
          </div>
        </div>

        {status.is_clean && (
          <Card>
            <CardContent className="py-8 text-center text-muted-foreground">
              Clean working tree
            </CardContent>
          </Card>
        )}

        <ScrollArea className="h-[calc(100vh-12rem)]">
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
                  onClick={() => handleFileClick(f.path)}
                  onStage={() => unstageFile(f.path)}
                />
              ))}
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
                  onClick={() => handleFileClick(f.path)}
                  onStage={() => stageFile(f.path)}
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
                  onClick={() => handleFileClick(f.path)}
                  onStage={() => stageFile(f.path)}
                />
              ))}
            </div>
          )}
        </ScrollArea>
      </div>

      {/* Diff panel */}
      {selectedFile && diff && (
        <div className="w-1/2 min-w-0 border-l pl-4">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold font-mono text-sm">
              {selectedFile}
            </h3>
            <div className="flex items-center gap-2">
              <span className="text-xs text-green-600">
                +{diff.total_additions}
              </span>
              <span className="text-xs text-red-600">
                -{diff.total_deletions}
              </span>
              <Button variant="ghost" size="sm" onClick={handleCloseDiff}>
                ✕
              </Button>
            </div>
          </div>
          <ScrollArea className="h-[calc(100vh-12rem)]">
            <pre className="text-xs font-mono leading-relaxed">
              {(diff.files || []).map((f, idx) => (
                <DiffBlock key={idx} file={f} />
              ))}
            </pre>
          </ScrollArea>
        </div>
      )}
    </div>
  );
}

interface FileRowProps {
  file: { path: string; staged: number; unstaged: number };
  type: "staged" | "unstaged" | "untracked";
  selected: boolean;
  onClick: () => void;
  onStage: () => void;
}

function FileRow({ file, type, selected, onClick, onStage }: FileRowProps) {
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

  return (
    <div
      className={cn(
        "flex items-center gap-2 px-3 py-1.5 border-l-2 cursor-pointer hover:bg-accent/50 text-sm transition-colors",
        colorClass,
        selected && "bg-accent"
      )}
      onClick={onClick}
    >
      <span
        className={cn(
          "w-5 text-center font-mono text-xs",
          type === "staged" && "text-green-500",
          type === "unstaged" && "text-yellow-500",
          type === "untracked" && "text-muted-foreground"
        )}
      >
        {label}
      </span>
      <span className="flex-1 truncate">{file.path}</span>
      <Button
        variant="ghost"
        size="sm"
        className="h-6 px-2 text-xs opacity-0 group-hover:opacity-100 hover:opacity-100"
        onClick={(e) => {
          e.stopPropagation();
          onStage();
        }}
      >
        {type === "staged" ? "Unstage" : "Stage"}
      </Button>
    </div>
  );
}

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
