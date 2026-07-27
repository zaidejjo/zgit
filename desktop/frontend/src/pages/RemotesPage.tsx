import { useEffect, useState } from "react";
import { GitBranch, Plus, Trash2, Globe } from "lucide-react";
import { useAppStore } from "@/store/app";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";

export default function RemotesPage() {
  const { remotes, loading, fetchRemotes, addRemote, removeRemote } = useAppStore();

  const [addOpen, setAddOpen] = useState(false);
  const [newName, setNewName] = useState("");
  const [newUrl, setNewUrl] = useState("");

  useEffect(() => {
    fetchRemotes();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleAdd = async () => {
    if (!newName.trim() || !newUrl.trim()) return;
    await addRemote(newName.trim(), newUrl.trim());
    setNewName("");
    setNewUrl("");
    setAddOpen(false);
  };

  const handleRemove = async (name: string) => {
    await removeRemote(name);
  };

  // Group remotes by name (each remote has fetch + push entries)
  const remoteList = remotes || [];
  const remoteMap = new Map<string, { name: string; fetchUrl?: string; pushUrl?: string }>();
  for (const r of remoteList) {
    const existing = remoteMap.get(r.name) || { name: r.name };
    if (r.type === "fetch" || !r.push_url) {
      existing.fetchUrl = r.url;
    }
    if (r.type === "push" || r.push_url) {
      existing.pushUrl = r.push_url || r.url;
    }
    remoteMap.set(r.name, existing);
  }
  const groupedRemotes = Array.from(remoteMap.values());

  return (
    <div className="h-full flex flex-col gap-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h2 className="text-xl font-bold">Remotes</h2>
          <span className="text-sm text-muted-foreground">
            {remoteList.length > 0 ? `${groupedRemotes.length} remote${groupedRemotes.length !== 1 ? "s" : ""}` : ""}
          </span>
        </div>
        <Dialog open={addOpen} onOpenChange={setAddOpen}>
          <DialogTrigger asChild>
            <Button size="sm" disabled={loading.remotes}>
              <Plus className="w-4 h-4 mr-1.5" />
              Add Remote
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Add Remote</DialogTitle>
            </DialogHeader>
            <div className="space-y-4 pt-2">
              <div className="space-y-2">
                <label className="text-sm font-medium">Name</label>
                <Input
                  placeholder="e.g. origin, upstream"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  onKeyDown={(e) => { if (e.key === "Enter") handleAdd(); }}
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">URL</label>
                <Input
                  placeholder="e.g. https://github.com/user/repo.git"
                  value={newUrl}
                  onChange={(e) => setNewUrl(e.target.value)}
                  onKeyDown={(e) => { if (e.key === "Enter") handleAdd(); }}
                />
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <Button variant="outline" onClick={() => setAddOpen(false)}>Cancel</Button>
                <Button onClick={handleAdd} disabled={!newName.trim() || !newUrl.trim()}>Add Remote</Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {/* Remote list */}
      <ScrollArea className="flex-1">
        {loading.remotes ? (
          <div className="flex items-center justify-center h-32 text-muted-foreground text-sm">
            Loading remotes...
          </div>
        ) : groupedRemotes.length === 0 ? (
          <Card>
            <CardContent className="py-12 text-center text-muted-foreground">
              <Globe className="w-10 h-10 mx-auto mb-3 opacity-40" />
              <p className="font-medium">No remotes configured</p>
              <p className="text-sm mt-1">
                Add a remote to push/pull changes
              </p>
              <Button
                variant="outline"
                size="sm"
                className="mt-4"
                onClick={() => setAddOpen(true)}
              >
                <Plus className="w-4 h-4 mr-1.5" />
                Add Remote
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-3">
            {groupedRemotes.map((remote) => (
              <Card key={remote.name}>
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-sm flex items-center gap-2">
                      <GitBranch className="w-4 h-4 text-muted-foreground" />
                      <span className="font-mono">{remote.name}</span>
                    </CardTitle>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 text-xs text-destructive hover:text-destructive hover:bg-destructive/10"
                      onClick={() => handleRemove(remote.name)}
                    >
                      <Trash2 className="w-3.5 h-3.5 mr-1" />
                      Remove
                    </Button>
                  </div>
                </CardHeader>
                <CardContent className="space-y-2 pt-0">
                  {remote.fetchUrl && (
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <span className="w-10 shrink-0 font-medium text-foreground/60">Fetch:</span>
                      <code className="flex-1 font-mono bg-muted/30 px-2 py-1 rounded truncate">
                        {remote.fetchUrl}
                      </code>
                    </div>
                  )}
                  {remote.pushUrl && remote.pushUrl !== remote.fetchUrl && (
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <span className="w-10 shrink-0 font-medium text-foreground/60">Push:</span>
                      <code className="flex-1 font-mono bg-muted/30 px-2 py-1 rounded truncate">
                        {remote.pushUrl}
                      </code>
                    </div>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </ScrollArea>
    </div>
  );
}
