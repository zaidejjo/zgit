import { useState } from "react";
import { History, Search, GitCommitHorizontal, AlignLeft } from "lucide-react";
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
    blame, fileHistory, historyPath, loading, repoPath,
    fetchBlame, fetchFileHistory,
  } = useAppStore();

  const [path, setPath] = useState(historyPath);
  const [follow, setFollow] = useState(true);
  const [pickaxe, setPickaxe] = useState("");
  const [pickaxeRe, setPickaxeRe] = useState("");
  const [blameRef, setBlameRef] = useState("");
  const [tab, setTab] = useState("history");

  const busy = !!loading.fileHistory || !!loading.blame;

  const runHistory = async () => {
    if (!path.trim() || busy) return;
    await fetchFileHistory(path.trim(), { follow, pickaxe: pickaxe.trim(), pickaxeRe: pickaxeRe.trim() });
  };

  const runBlame = async () => {
    if (!path.trim() || busy) return;
    await fetchBlame(path.trim(), blameRef.trim());
  };

  const search = () => {
    if (tab === "blame") void runBlame();
    else void runHistory();
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
        <input
          className="flex h-9 flex-1 min-w-[200px] rounded-md border border-input bg-background px-3 py-2 text-sm font-mono ring-offset-background placeholder:text-muted-foreground placeholder:font-sans focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          placeholder="File path, e.g. src/main.go"
          value={path}
          onChange={(e) => setPath(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && search()}
        />
        <Button size="sm" onClick={search} disabled={!path.trim() || busy}>
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
