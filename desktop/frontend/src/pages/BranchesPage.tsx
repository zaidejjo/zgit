import { useEffect, useState } from "react";
import { useAppStore } from "@/store/app";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";

export default function BranchesPage() {
  const { branches, currentBranch, loading, fetchBranches, checkoutBranch, createBranch, deleteBranch } = useAppStore();
  const [newBranchName, setNewBranchName] = useState("");
  const [showCreate, setShowCreate] = useState(false);

  useEffect(() => {
    fetchBranches();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (loading.branches && branches.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        Loading branches...
      </div>
    );
  }

  const localBranches = branches.filter((b) => !b.Name.startsWith("remotes/"));
  const remoteBranches = branches.filter((b) => b.Name.startsWith("remotes/"));

  const handleCreate = async () => {
    if (!newBranchName.trim()) return;
    await createBranch(newBranchName.trim());
    setNewBranchName("");
    setShowCreate(false);
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-bold">Branches</h2>
        <Button variant="outline" size="sm" onClick={() => setShowCreate(!showCreate)}>
          {showCreate ? "Cancel" : "New Branch"}
        </Button>
      </div>

      {showCreate && (
        <Card className="mb-4">
          <CardContent className="pt-4">
            <div className="flex gap-2">
              <input
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                placeholder="Branch name"
                value={newBranchName}
                onChange={(e) => setNewBranchName(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleCreate()}
                autoFocus
              />
              <Button onClick={handleCreate}>Create</Button>
            </div>
          </CardContent>
        </Card>
      )}

      <ScrollArea className="h-[calc(100vh-12rem)]">
        {/* Local branches */}
        <div className="mb-6">
          <h3 className="text-sm font-medium text-muted-foreground mb-2">
            Local ({localBranches.length})
          </h3>
          {localBranches.map((branch) => (
            <div
              key={branch.Name}
              className={cn(
                "flex items-center gap-3 px-3 py-2 rounded-lg transition-colors",
                branch.Current
                  ? "bg-primary/10 border border-primary/20"
                  : "hover:bg-accent/50 cursor-pointer"
              )}
              onClick={() => !branch.Current && checkoutBranch(branch.Name)}
            >
              <span className={branch.Current ? "text-primary" : "text-muted-foreground"}>
                {branch.Current ? "●" : "○"}
              </span>
              <span className="flex-1 font-medium font-mono text-sm">{branch.Name}</span>
              {branch.Current && (
                <Badge variant="secondary" className="text-xs">Current</Badge>
              )}
              {branch.Upstream && (
                <span className="text-xs text-muted-foreground">{branch.Upstream}</span>
              )}
              {(branch.Ahead > 0 || branch.Behind > 0) && (
                <span className="text-xs text-muted-foreground">
                  {branch.Ahead > 0 && <span className="text-green-500">+{branch.Ahead}</span>}
                  {branch.Behind > 0 && <span className="text-red-500"> -{branch.Behind}</span>}
                </span>
              )}
              {!branch.Current && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-6 px-2 text-xs opacity-0 hover:opacity-100"
                  onClick={(e) => {
                    e.stopPropagation();
                    deleteBranch(branch.Name, false);
                  }}
                >
                  Delete
                </Button>
              )}
            </div>
          ))}
        </div>

        <Separator className="my-4" />

        {/* Remote branches */}
        {remoteBranches.length > 0 && (
          <div>
            <h3 className="text-sm font-medium text-muted-foreground mb-2">
              Remote ({remoteBranches.length})
            </h3>
            {remoteBranches.map((branch) => (
              <div
                key={branch.Name}
                className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-accent/50 transition-colors"
              >
                <span className="text-muted-foreground">◯</span>
                <span className="flex-1 font-mono text-sm">{branch.Name}</span>
              </div>
            ))}
          </div>
        )}
      </ScrollArea>
    </div>
  );
}
