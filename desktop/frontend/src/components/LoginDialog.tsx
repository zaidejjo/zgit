import { useState } from "react";
import { useAppStore } from "@/store/app";
import { Button } from "@/components/ui/button";

export default function LoginDialog() {
  const { loginDialogOpen, setLoginDialogOpen, authenticateGitHub, loading } = useAppStore();
  const [token, setToken] = useState("");
  const [showToken, setShowToken] = useState(false);

  if (!loginDialogOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token.trim()) return;
    await authenticateGitHub(token.trim());
  };

  const handleClose = () => {
    setLoginDialogOpen(false);
    setToken("");
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-full max-w-md rounded-lg border bg-card p-6 shadow-lg">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">GitHub Authentication</h2>
          <button
            className="text-muted-foreground hover:text-foreground"
            onClick={handleClose}
          >
            ✕
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="space-y-4">
            <div className="text-sm text-muted-foreground space-y-2">
              <p>
                Enter a GitHub Personal Access Token (PAT) to enable PR, Issue,
                and repository features.
              </p>
              <p>
                Create one at{" "}
                <a
                  href="https://github.com/settings/tokens"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-primary underline underline-offset-2"
                >
                  github.com/settings/tokens
                </a>
                {" "}with <code className="text-xs bg-muted px-1 py-0.5 rounded">repo</code> and{" "}
                <code className="text-xs bg-muted px-1 py-0.5 rounded">read:user</code> scopes.
              </p>
            </div>

            <div className="relative">
              <input
                type={showToken ? "text" : "password"}
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder="ghp_xxxxxxxxxxxxxxxxxxxx"
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 pr-10 text-sm font-mono ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                autoFocus
                disabled={loading.auth}
              />
              <button
                type="button"
                className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground text-xs"
                onClick={() => setShowToken(!showToken)}
                tabIndex={-1}
              >
                {showToken ? "Hide" : "Show"}
              </button>
            </div>

            <div className="flex justify-end gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleClose}
                disabled={loading.auth}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                size="sm"
                disabled={!token.trim() || loading.auth}
              >
                {loading.auth ? "Authenticating..." : "Sign In"}
              </Button>
            </div>
          </div>
        </form>
      </div>
    </div>
  );
}
