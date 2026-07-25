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

  const stagedFiles = status.Files.filter((f) => f.X !== " " && f.X !== "?" && f.X !== "!");
  const unstagedFiles = status.Files.filter((f) => f.Y !== " " && f.X !== " " && f.X !== "?" && f.X !== "!");
  const untrackedFiles = status.Files.filter((f) => f.X === "?" || f.X === "!");

  return (
    <div className="flex gap-4 h-full">
      {/* File list */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <h2 className="text-xl font-bold">
              Status
            </h2>
            {status.Branch && (
              <Badge variant="outline" className="font-mono">
                {status.Branch}
              </Badge>
            )}
            {!status.IsClean && (
              <Badge variant="warning">
                {status.Ahead > 0 ? `+${status.Ahead}` : ""}
                {status.Behind > 0 ? ` -${status.Behind}` : ""}
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

        {status.IsClean && (
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
                  key={f.Path}
                  file={f}
                  type="staged"
                  selected={selectedFile === f.Path}
                  onClick={() => handleFileClick(f.Path)}
                  onStage={() => unstageFile(f.Path)}
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
                  key={f.Path}
                  file={f}
                  type="unstaged"
                  selected={selectedFile === f.Path}
                  onClick={() => handleFileClick(f.Path)}
                  onStage={() => stageFile(f.Path)}
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
                  key={f.Path}
                  file={f}
                  type="untracked"
                  selected={selectedFile === f.Path}
                  onClick={() => handleFileClick(f.Path)}
                  onStage={() => stageFile(f.Path)}
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
                +{diff.TotalAdds}
              </span>
              <span className="text-xs text-red-600">
                -{diff.TotalDeletes}
              </span>
              <Button variant="ghost" size="sm" onClick={handleCloseDiff}>
                ✕
              </Button>
            </div>
          </div>
          <ScrollArea className="h-[calc(100vh-12rem)]">
            <pre className="text-xs font-mono leading-relaxed">
              {diff.Files.map((f, idx) => (
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
  file: { Path: string; X: string; Y: string };
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
      <span className="flex-1 truncate">{file.Path}</span>
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

function DiffBlock({ file }: { file: { NewPath: string; UnifiedDiff: string; Additions: number; Deletions: number } }) {
  if (!file.UnifiedDiff) {
    return (
      <div className="py-4 text-muted-foreground italic">
        No diff content available
      </div>
    );
  }

  return (
    <div className="mb-4">
      <div className="flex gap-2 text-xs mb-1">
        <span className="text-green-500">+{file.Additions}</span>
        <span className="text-red-500">-{file.Deletions}</span>
        <span className="text-muted-foreground">{file.NewPath}</span>
      </div>
      <div className="bg-muted/30 rounded p-2 overflow-x-auto">
        <code className="text-xs">
          {file.UnifiedDiff.split("\n").map((line, i) => {
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
