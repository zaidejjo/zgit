import { useEffect } from "react";
import {
  FolderOpen, GitBranch, Globe, GitCommitHorizontal, Settings, User, Calendar,
} from "lucide-react";
import { useAppStore } from "@/store/app";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";

export default function SettingsPage() {
  const {
    repoPath, status, currentBranch,
    repository, fetchRepository,
    ghAuthenticated, ghUser,
  } = useAppStore();

  useEffect(() => {
    fetchRepository();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const repoName = repoPath ? repoPath.split("/").pop() || repoPath : "No repository";

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <div className="flex items-center gap-2">
        <Settings className="w-5 h-5" />
        <h2 className="text-xl font-bold">Repository Settings</h2>
      </div>

      {/* Repository info */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <FolderOpen className="w-4 h-4 text-muted-foreground" />
            Repository
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center gap-2 text-sm">
            <span className="w-24 text-muted-foreground shrink-0">Name:</span>
            <span className="font-medium">{repoName}</span>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <span className="w-24 text-muted-foreground shrink-0">Path:</span>
            <code className="font-mono text-xs bg-muted/30 px-2 py-1 rounded truncate">
              {repoPath || "—"}
            </code>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <span className="w-24 text-muted-foreground shrink-0">Current branch:</span>
            <Badge variant="outline" className="font-mono">
              {currentBranch || status?.branch || "—"}
            </Badge>
          </div>
        </CardContent>
      </Card>

      {/* GitHub info */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <Globe className="w-4 h-4 text-muted-foreground" />
            GitHub Connection
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {ghAuthenticated && ghUser ? (
            <>
              <div className="flex items-center gap-3">
                <img
                  src={ghUser.avatar_url}
                  alt={ghUser.login}
                  className="w-10 h-10 rounded-full"
                />
                <div>
                  <p className="font-medium">{ghUser.name || ghUser.login}</p>
                  <p className="text-sm text-muted-foreground">{ghUser.login}</p>
                </div>
              </div>
              {repository && (
                <>
                  <Separator />
                  <div className="flex items-center gap-2 text-sm">
                    <span className="w-24 text-muted-foreground shrink-0">Repository:</span>
                    <span>{repository.full_name || repository.name || "—"}</span>
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    <span className="w-24 text-muted-foreground shrink-0">Default branch:</span>
                    <Badge variant="outline" className="font-mono text-xs">
                      {repository.default_branch || "—"}
                    </Badge>
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    <span className="w-24 text-muted-foreground shrink-0">Visibility:</span>
                    <span>{repository.is_private ? "Private" : "Public"}</span>
                  </div>
                  {repository.description && (
                    <p className="text-sm text-muted-foreground mt-2">
                      {repository.description}
                    </p>
                  )}
                </>
              )}
            </>
          ) : (
            <div className="flex items-center gap-3 py-2">
              <User className="w-8 h-8 text-muted-foreground" />
              <div>
                <p className="text-sm font-medium">Not connected</p>
                <p className="text-xs text-muted-foreground">Sign in to GitHub from the header</p>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* About */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <GitCommitHorizontal className="w-4 h-4 text-muted-foreground" />
            About
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm text-muted-foreground">
          <p>zgit — Git & GitHub desktop client</p>
          <div className="flex items-center gap-2">
            <Calendar className="w-3.5 h-3.5" />
            <span>Built with Go + React + Wails</span>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
