import { useEffect, useState } from "react";
import { GitBranch, Plus, Trash2, Pencil, Globe, Link } from "lucide-react";
import { useAppStore } from "@/store/app";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";

export default function RemotesPage() {
  const { remotes, loading, fetchRemotes, addRemote, removeRemote, renameRemote, setRemoteUrl } = useAppStore();

  const [addOpen, setAddOpen] = useState(false);
  const [newName, setNewName] = useState("");
  const [newUrl, setNewUrl] = useState("");

  const [renameTarget, setRenameTarget] = useState<string | null>(null);
  const [renameName, setRenameName] = useState("");
  const [renameOpen, setRenameOpen] = useState(false);

  const [editUrlTarget, setEditUrlTarget] = useState<string | null>(null);
  const [editUrl, setEditUrl] = useState("");
  const [editUrlOpen, setEditUrlOpen] = useState(false);

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

  const handleRenameStart = (name: string) => {
    setRenameTarget(name);
    setRenameName(name);
    setRenameOpen(true);
  };

  const handleRename = async () => {
    if (!renameTarget || !renameName.trim() || renameName.trim() === renameTarget) return;
    await renameRemote(renameTarget, renameName.trim());
    setRenameOpen(false);
    setRenameTarget(null);
    setRenameName("");
  };

  const handleEditUrlStart = (name: string, currentUrl: string) => {
    setEditUrlTarget(name);
    setEditUrl(currentUrl);
    setEditUrlOpen(true);
  };

  const handleEditUrl = async () => {
    if (!editUrlTarget || !editUrl.trim()) return;
    await setRemoteUrl(editUrlTarget, editUrl.trim());
    setEditUrlOpen(false);
    setEditUrlTarget(null);
    setEditUrl("");
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

      {/* Rename Remote dialog */}
      <Dialog open={renameOpen} onOpenChange={(open) => { if (!open) { setRenameOpen(false); setRenameTarget(null); setRenameName(""); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rename Remote</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            <div className="space-y-2">
              <label className="text-sm font-medium">New name</label>
              <Input
                placeholder="e.g. upstream"
                value={renameName}
                onChange={(e) => setRenameName(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") handleRename(); }}
                autoFocus
              />
            </div>
            <p className="text-xs text-muted-foreground">
              Rename <code className="font-mono bg-muted/30 px-1 rounded">{renameTarget}</code> to a new name.
            </p>
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="outline" onClick={() => { setRenameOpen(false); setRenameTarget(null); setRenameName(""); }}>Cancel</Button>
              <Button onClick={handleRename} disabled={!renameName.trim() || renameName.trim() === renameTarget}>Rename</Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Edit Remote URL dialog */}
      <Dialog open={editUrlOpen} onOpenChange={(open) => { if (!open) { setEditUrlOpen(false); setEditUrlTarget(null); setEditUrl(""); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Remote URL</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            <div className="space-y-2">
              <label className="text-sm font-medium">New URL</label>
              <Input
                placeholder="e.g. https://github.com/user/repo.git"
                value={editUrl}
                onChange={(e) => setEditUrl(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") handleEditUrl(); }}
                autoFocus
              />
            </div>
            <p className="text-xs text-muted-foreground">
              Update URL for <code className="font-mono bg-muted/30 px-1 rounded">{editUrlTarget}</code>.
            </p>
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="outline" onClick={() => { setEditUrlOpen(false); setEditUrlTarget(null); setEditUrl(""); }}>Cancel</Button>
              <Button onClick={handleEditUrl} disabled={!editUrl.trim()}>Save</Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

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
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-7 text-xs text-muted-foreground hover:text-foreground"
                        onClick={() => handleRenameStart(remote.name)}
                      >
                        <Pencil className="w-3.5 h-3.5 mr-1" />
                        Rename
                      </Button>
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
                  </div>
                </CardHeader>
                <CardContent className="space-y-2 pt-0">
                  {remote.fetchUrl && (
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <span className="w-10 shrink-0 font-medium text-foreground/60">Fetch:</span>
                      <code className="flex-1 font-mono bg-muted/30 px-2 py-1 rounded truncate">
                        {remote.fetchUrl}
                      </code>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-6 text-xs shrink-0 text-muted-foreground hover:text-foreground"
                        onClick={() => handleEditUrlStart(remote.name, remote.fetchUrl || "")}
                      >
                        <Link className="w-3 h-3 mr-1" />
                        Edit URL
                      </Button>
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
