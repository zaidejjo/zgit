import { useState, useEffect, useRef, useCallback } from "react";
import { useAppStore } from "@/store/app";
import { Button } from "@/components/ui/button";
import { BrowserOpenURL } from "../../wailsjs/runtime";

type AuthMode = "pat" | "device";

export default function LoginDialog() {
  const {
    loginDialogOpen, setLoginDialogOpen, authenticateGitHub,
    startDeviceFlow, pollDeviceFlow, loading,
  } = useAppStore();

  const [mode, setMode] = useState<AuthMode>("pat");
  const [token, setToken] = useState("");
  const [showToken, setShowToken] = useState(false);

  // Device flow state
  const [deviceCode, setDeviceCode] = useState<{
    user_code: string;
    device_code: string;
    interval: number;
  } | null>(null);
  const [pollStatus, setPollStatus] = useState<string>("");
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Cleanup polling on unmount
  useEffect(() => {
    return () => {
      if (pollingRef.current) clearInterval(pollingRef.current);
    };
  }, []);

  const stopPolling = useCallback(() => {
    if (pollingRef.current) {
      clearInterval(pollingRef.current);
      pollingRef.current = null;
    }
  }, []);

  if (!loginDialogOpen) return null;

  // --- PAT mode ---

  const handlePatSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token.trim()) return;
    const ok = await authenticateGitHub(token.trim());
    if (ok) {
      setToken("");
      setLoginDialogOpen(false);
    }
  };

  // --- Device flow mode ---

  const handleStartDeviceFlow = async () => {
    const code = await startDeviceFlow();
    if (!code) return;

    setDeviceCode(code);
    setPollStatus("Waiting for authorization...");

    // Open browser to verification URL
    BrowserOpenURL("https://github.com/login/device");

    // Start polling
    const intervalMs = (code.interval || 5) * 1000;

    pollingRef.current = setInterval(async () => {
      const token = await pollDeviceFlow(code.device_code);
      if (token) {
        // Got the token — authenticate
        stopPolling();
        setPollStatus("Authorized! Saving...");
        const ok = await authenticateGitHub(token);
        if (ok) {
          setDeviceCode(null);
          setLoginDialogOpen(false);
        } else {
          setPollStatus("Failed to save token. Try again.");
        }
      }
      // If token is empty, keep polling (authorization_pending)
    }, intervalMs);
  };

  const handleCancelDeviceFlow = () => {
    stopPolling();
    setDeviceCode(null);
    setPollStatus("");
  };

  const handleCopyCode = () => {
    if (deviceCode?.user_code) {
      navigator.clipboard.writeText(deviceCode.user_code);
    }
  };

  const handleClose = () => {
    stopPolling();
    setDeviceCode(null);
    setPollStatus("");
    setToken("");
    setLoginDialogOpen(false);
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

        {/* Mode tabs */}
        <div className="flex gap-1 mb-4 border-b pb-2">
          <button
            className={`px-3 py-1 text-sm rounded-t transition-colors ${
              mode === "pat"
                ? "bg-primary text-primary-foreground"
                : "text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => { setMode("pat"); setDeviceCode(null); stopPolling(); }}
          >
            Personal Access Token
          </button>
          <button
            className={`px-3 py-1 text-sm rounded-t transition-colors ${
              mode === "device"
                ? "bg-primary text-primary-foreground"
                : "text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => { setMode("device"); }}
          >
            Sign in via Browser
          </button>
        </div>

        {mode === "pat" ? (
          /* --- PAT mode --- */
          <form onSubmit={handlePatSubmit}>
            <div className="space-y-4">
              <div className="text-sm text-muted-foreground space-y-2">
                <p>
                  Enter a GitHub Personal Access Token (PAT) to enable PR, Issue,
                  and repository features.
                </p>
                <p>
                  Create one at{" "}
                  <a
                    href="#"
                    onClick={(e) => {
                      e.preventDefault();
                      BrowserOpenURL("https://github.com/settings/tokens");
                    }}
                    className="text-primary underline underline-offset-2 cursor-pointer"
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
        ) : (
          /* --- Device flow mode --- */
          <div className="space-y-4">
            <div className="text-sm text-muted-foreground space-y-2">
              <p>
                Authorize zgit via your browser. No token needed — just click
                the button below and confirm the code.
              </p>
            </div>

            {!deviceCode ? (
              /* Initial state: show start button */
              <div className="flex flex-col items-center gap-4 py-4">
                <div className="text-4xl">🔑</div>
                <p className="text-sm text-muted-foreground text-center">
                  GitHub will ask you to enter a code and confirm access to your
                  repositories.
                </p>
                <Button
                  size="sm"
                  onClick={handleStartDeviceFlow}
                  disabled={loading.auth}
                >
                  {loading.auth ? "Starting..." : "Sign in via Browser"}
                </Button>
              </div>
            ) : (
              /* Device code displayed */
              <div className="space-y-4">
                <div className="text-center">
                  <p className="text-xs text-muted-foreground mb-1">
                    Enter this code on GitHub:
                  </p>
                  <div className="text-3xl font-mono font-bold tracking-widest bg-muted px-4 py-3 rounded-lg select-all">
                    {deviceCode.user_code}
                  </div>
                </div>

                <div className="flex justify-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleCopyCode}
                  >
                    Copy Code
                  </Button>
                  <Button
                    size="sm"
                    onClick={() => BrowserOpenURL("https://github.com/login/device")}
                  >
                    Open Browser
                  </Button>
                </div>

                <div className="text-center">
                  <p className="text-sm text-muted-foreground">{pollStatus}</p>
                  {pollStatus && !pollStatus.includes("Authorized") && !pollStatus.includes("Failed") && (
                    <div className="mt-2 flex justify-center">
                      <div className="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin" />
                    </div>
                  )}
                </div>

                <div className="flex justify-center">
                  <button
                    className="text-xs text-muted-foreground hover:text-foreground underline"
                    onClick={handleCancelDeviceFlow}
                  >
                    Cancel
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
