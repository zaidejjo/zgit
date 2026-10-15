import { useEffect, useMemo, useRef, useState } from "react";
import { History, Search, GitCommitHorizontal, AlignLeft, FileText } from "lucide-react";
import { useAppStore } from "@/store/app";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Checkbox } from "@/components/ui/checkbox";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import EmptyState from "@/components/EmptyState";

export default function HistoryPage() {
  const {
    blame, fileHistory, historyPath, loading, repoPath, repoFiles,
    fetchBlame, fetchFileHistory, fetchRepoFiles,
  } = useAppStore();

  const [path, setPath] = useState(historyPath);
  const [follow, setFollow] = useState(true);
  const [pickaxe, setPickaxe] = useState("");
  const [pickaxeRe, setPickaxeRe] = useState("");
  const [blameRef, setBlameRef] = useState("");
  const [tab, setTab] = useState("history");
  const [suggestOpen, setSuggestOpen] = useState(false);
  const [suggestIdx, setSuggestIdx] = useState(0);
  const blurTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    fetchRepoFiles();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [repoPath]);

  useEffect(() => () => {
    if (blurTimer.current) clearTimeout(blurTimer.current);
  }, []);

  // Substring match (case-insensitive), basename hits first, capped at 8.
  const suggestions = useMemo(() => {
    const q = path.trim().toLowerCase();
    if (!q || repoFiles.length === 0) return [];
    const scored = repoFiles
      .map((f) => {
        const lower = f.toLowerCase();
        const idx = lower.indexOf(q);
        if (idx < 0) return null;
        const base = lower.split("/").pop() || "";
        const baseHit = base.includes(q) ? 0 : 1;
        return { f, score: baseHit * 10000 + idx };
      })
      .filter((s): s is { f: string; score: number } => s !== null)
      .sort((a, b) => a.score - b.score)
      .slice(0, 8)
      .map((s) => s.f);
    return scored;
  }, [path, repoFiles]);

  const chooseSuggestion = (f: string) => {
    setPath(f);
    setSuggestOpen(false);
    setSuggestIdx(0);
  };

  const busy = !!loading.fileHistory || !!loading.blame;

  const runHistory = async (p = path) => {
    if (!p.trim() || busy) return;
    await fetchFileHistory(p.trim(), { follow, pickaxe: pickaxe.trim(), pickaxeRe: pickaxeRe.trim() });
  };

  const runBlame = async (p = path) => {
    if (!p.trim() || busy) return;
    await fetchBlame(p.trim(), blameRef.trim());
  };

  const search = (p = path) => {
    if (tab === "blame") void runBlame(p);
    else void runHistory(p);
  };

  return (
    <div className="h-full flex flex-col gap-3">
      <div className="flex items-center gap-2">
        <h2 className="text-xl font-bold">History</h2>
        {historyPath && (
          <Badge variant="outline" className="font-mono text-xs max-w-[300px] truncate">
            {historyPath}
          </Badge>
        )}
      </div>

      {/* Query bar */}
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative flex-1 min-w-[200px]">
          <input
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono ring-offset-background placeholder:text-muted-foreground placeholder:font-sans focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            placeholder="File path, e.g. src/main.go"
            value={path}
            onChange={(e) => { setPath(e.target.value); setSuggestOpen(true); setSuggestIdx(0); }}
            onFocus={() => { if (path.trim()) setSuggestOpen(true); }}
            onBlur={() => {
              blurTimer.current = setTimeout(() => setSuggestOpen(false), 120);
            }}
            onKeyDown={(e) => {
              if (suggestOpen && suggestions.length > 0) {
                if (e.key === "ArrowDown") {
                  e.preventDefault();
                  setSuggestIdx((i) => (i + 1) % suggestions.length);
                  return;
                }
                if (e.key === "ArrowUp") {
                  e.preventDefault();
                  setSuggestIdx((i) => (i - 1 + suggestions.length) % suggestions.length);
                  return;
                }
                if (e.key === "Tab") {
                  e.preventDefault();
                  chooseSuggestion(suggestions[suggestIdx]);
                  return;
                }
                if (e.key === "Escape") {
                  setSuggestOpen(false);
                  return;
                }
              }
              if (e.key === "Enter") {
                if (suggestOpen && suggestions.length > 0) {
                  const f = suggestions[suggestIdx];
                  chooseSuggestion(f);
                  search(f);
                } else {
                  search();
                }
              }
            }}
          />
          {suggestOpen && suggestions.length > 0 && (
            <div className="absolute left-0 right-0 top-full mt-1 z-50 rounded-lg border border-border/60 bg-popover shadow-xl py-1 max-h-64 overflow-y-auto">
              {suggestions.map((f, i) => (
                <button
                  key={f}
                  className={cn(
                    "flex items-center gap-2 w-full px-3 py-1.5 text-left font-mono text-xs transition-colors",
                    i === suggestIdx ? "bg-accent text-accent-foreground" : "hover:bg-accent/50"
                  )}
                  onMouseDown={(e) => {
                    e.preventDefault();
                    chooseSuggestion(f);
                    setTimeout(() => search(f), 0);
                  }}
                  onMouseEnter={() => setSuggestIdx(i)}
                >
                  <FileText className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                  <span className="truncate">{f}</span>
                </button>
              ))}
            </div>
          )}
        </div>
        <Button size="sm" onClick={() => search()} disabled={!path.trim() || busy}>
          <Search className="w-4 h-4 mr-1.5" />
          {busy ? "Loading..." : "Search"}
        </Button>
      </div>

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="history">
            <GitCommitHorizontal className="w-3.5 h-3.5 mr-1.5" />
            Commits
          </TabsTrigger>
          <TabsTrigger value="blame">
            <AlignLeft className="w-3.5 h-3.5 mr-1.5" />
            Blame
          </TabsTrigger>
        </TabsList>

        <TabsContent value="history" className="mt-3">
          <div className="flex flex-wrap items-center gap-4 mb-3 text-sm">
            <label className="flex items-center gap-2 cursor-pointer text-muted-foreground">
              <Checkbox checked={follow} onChange={() => setFollow(!follow)} />
              <span>Follow renames (<span className="font-mono">--follow</span>)</span>
            </label>
            <input
              className="h-8 rounded-md border border-input bg-background px-2.5 text-xs font-mono w-44 ring-offset-background placeholder:text-muted-foreground placeholder:font-sans focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              placeholder="-S pickaxe string"
              value={pickaxe}
              onChange={(e) => setPickaxe(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && runHistory()}
            />
            <input
              className="h-8 rounded-md border border-input bg-background px-2.5 text-xs font-mono w-44 ring-offset-background placeholder:text-muted-foreground placeholder:font-sans focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              placeholder="-G pickaxe regex"
              value={pickaxeRe}
              onChange={(e) => setPickaxeRe(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && runHistory()}
            />
          </div>
          {fileHistory.length === 0 ? (
            <EmptyState
              icon={<History className="w-16 h-16" />}
              title="No file history yet"
              description={repoPath ? "Enter a file path above to see every commit that touched it." : "Open a repository first."}
            />
          ) : (
            <ScrollArea className="h-[calc(100vh-19rem)]">
              <div className="space-y-0.5">
                {fileHistory.map((c) => (
                  <div key={c.hash} className="flex items-start gap-3 px-3 py-2 rounded-lg hover:bg-accent/50 transition-colors">
                    <span className="font-mono text-xs text-primary shrink-0 mt-0.5">{c.hash.slice(0, 8)}</span>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm truncate">{c.message}</p>
                      <p className="text-xs text-muted-foreground truncate">
                        {c.author} · {new Date(c.timestamp).toLocaleDateString()}
                        {c.ref_names ? ` · ${c.ref_names}` : ""}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            </ScrollArea>
          )}
        </TabsContent>

        <TabsContent value="blame" className="mt-3">
          <div className="flex flex-wrap items-center gap-2 mb-3">
            <input
              className="h-8 rounded-md border border-input bg-background px-2.5 text-xs font-mono w-44 ring-offset-background placeholder:text-muted-foreground placeholder:font-sans focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              placeholder="Revision (default: HEAD)"
              value={blameRef}
              onChange={(e) => setBlameRef(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && runBlame()}
            />
          </div>
          {blame.length === 0 ? (
            <EmptyState
              icon={<AlignLeft className="w-16 h-16" />}
              title="No blame yet"
              description={repoPath ? "Enter a file path above to see who changed each line." : "Open a repository first."}
            />
          ) : (
            <ScrollArea className="h-[calc(100vh-19rem)]">
              <div className="font-mono text-xs">
                {blame.map((l) => (
                  <div key={l.line_no} className="flex gap-3 px-2 py-0.5 rounded hover:bg-accent/50 transition-colors">
                    <span className="w-10 shrink-0 text-right text-muted-foreground select-none">{l.line_no}</span>
                    <span className={cn("w-20 shrink-0 truncate", l.is_uncommitted ? "text-yellow-500" : "text-primary")} title={`${l.hash} · ${l.summary}`}>
                      {l.is_uncommitted ? "uncomm." : l.hash.slice(0, 8)}
                    </span>
                    <span className="w-28 shrink-0 truncate text-muted-foreground" title={l.email}>{l.author}</span>
                    <span className="flex-1 whitespace-pre-wrap break-all">{l.content}</span>
                  </div>
                ))}
              </div>
            </ScrollArea>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}
