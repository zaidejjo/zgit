import { useState, useEffect, useRef, useCallback } from "react";
import { useAppStore } from "@/store/app";
import { cn } from "@/lib/utils";
// OpenURL is a custom non-blocking Go binding — avoids xdg-open blocking the Wails bridge on Linux.
// We import from the generated App binding module instead of the Wails runtime for this reason.
// eslint-disable-next-line @typescript-eslint/no-unused-vars
import { OpenURL } from "../../wailsjs/go/main/App";

type AuthMode = "device" | "pat";

/* ─── Step states for device flow ─── */
type DeviceStep =
  | "idle"         // not started
  | "starting"     // calling StartDeviceFlow backend
  | "awaiting"     // code displayed, polling GitHub
  | "authenticating" // got token, saving via AuthenticateGitHub
  | "success"      // done
  | "error";       // something failed

export default function LoginDialog() {
  const {
    loginDialogOpen, setLoginDialogOpen,
    authenticateGitHub, saveGitHubToken, validateGitHubToken,
    startDeviceFlow,
    pollDeviceFlowWithRetry, cancelDeviceFlow,
    setSuccessMessage,
  } = useAppStore();

  const [mode, setMode] = useState<AuthMode>("device");

  // Device flow state
  const [deviceStep, setDeviceStep] = useState<DeviceStep>("idle");
  const [deviceCode, setDeviceCode] = useState<{
    user_code: string;
    device_code: string;
    verification_uri: string;
    interval: number;
  } | null>(null);
  const [deviceError, setDeviceError] = useState<string>("");
  const [copied, setCopied] = useState(false);

  // PAT state
  const [token, setToken] = useState("");
  const [showToken, setShowToken] = useState(false);
  const [patError, setPatError] = useState("");
  const [patLoading, setPatLoading] = useState(false);

  // Track whether component is still mounted (cancel polling on unmount)
  const mountedRef = useRef(true);
  useEffect(() => {
    return () => { mountedRef.current = false; cancelDeviceFlow(); };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  /* ─── Reset everything on dialog close ─── */
  const resetAll = useCallback(() => {
    cancelDeviceFlow();
    setDeviceStep("idle");
    setDeviceCode(null);
    setDeviceError("");
    setCopied(false);
    setToken("");
    setShowToken(false);
    setPatError("");
    setPatLoading(false);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  /* ─── Dialog close handler ─── */
  const handleClose = () => {
    resetAll();
    setLoginDialogOpen(false);
  };

  if (!loginDialogOpen) return null;

  /* ════════════════ Device flow ════════════════ */

  const handleStartDeviceFlow = async () => {
    setDeviceStep("starting");
    setDeviceError("");

    try {
      const code = await startDeviceFlow();
      if (!code) {
        setDeviceStep("error");
        setDeviceError("Failed to start device flow. Is your network connected?");
        return;
      }

      setDeviceCode(code);
      setDeviceStep("awaiting");

      // Open browser to the GitHub device activation page (non-blocking)
      OpenURL(code.verification_uri || "https://github.com/login/device");

      // Blocking poll — handles interval, slow_down, authorization_pending internally.
      // Go backend polls at the correct interval, adds +5s on slow_down, gives up after 10min.
      // CancelDeviceFlow() on the Go side cancels the underlying context.
      try {
        const accessToken = await pollDeviceFlowWithRetry(code.device_code, code.interval || 5);
        if (!mountedRef.current) return;

        console.log("[LoginDialog] Device flow token received:", accessToken.slice(0, 8) + "...");
        setDeviceStep("authenticating");

        // Save the token INSTANTLY — no validation call, no hang risk.
        // This closes the dialog and shows success toast immediately.
        const saved = await saveGitHubToken(accessToken);
        if (!mountedRef.current) return;

        if (!saved) {
          setDeviceStep("error");
          setDeviceError("Token received but could not be saved to config. Check permissions and try again.");
          return;
        }

        console.log("[LoginDialog] Token saved, closing dialog");
        setDeviceStep("success");
        setSuccessMessage("Connected to GitHub successfully!");
        handleClose(); // Close immediately — no setTimeout delay

        // Fetch user profile in the background — dialog is already closed.
        // Even if this fails, the user is still authenticated.
        console.log("[LoginDialog] Fetching user profile in background...");
        validateGitHubToken().then((user) => {
          console.log("[LoginDialog] User profile:", user?.login || "unavailable");
          // Trigger data refresh
          const store = useAppStore.getState();
          store.fetchPullRequests?.().catch(() => {});
          store.fetchIssues?.().catch(() => {});
        }).catch((err) => {
          console.warn("[LoginDialog] Background user fetch failed:", err);
        });
      } catch (err: any) {
        if (!mountedRef.current) return;
        console.error("[LoginDialog] Poll/device flow error:", err);
        // Check if cancellation was intentional
        if (err?.message?.includes("canceled") || err?.message?.includes("Cancelled") || err?.message?.includes("context canceled")) {
          resetAll();
          return;
        }
        setDeviceStep("error");
        setDeviceError(err?.message || "Authorization failed. Please try again.");
      }
    } catch (err: any) {
      if (!mountedRef.current) return;
      setDeviceStep("error");
      setDeviceError(err?.message || "Failed to start device flow.");
    }
  };

  const handleRetryDevice = () => {
    resetAll();
    setMode("device");
  };

  const handleCancelDevicePoll = () => {
    cancelDeviceFlow();
    resetAll();
  };

  /* ════════════════ PAT flow ════════════════ */

  const handlePatSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token.trim()) return;
    setPatLoading(true);
    setPatError("");

    try {
      const ok = await authenticateGitHub(token.trim());
      if (ok) {
        setSuccessMessage("Connected to GitHub successfully!");
        handleClose();
      } else {
        setPatError("Authentication failed. Check that your token has the correct scopes (repo, read:user).");
      }
    } catch (err: any) {
      setPatError(err?.message || "Authentication failed. Check your token and try again.");
    } finally {
      setPatLoading(false);
    }
  };

  /* ════════════════ Render ════════════════ */

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="w-full max-w-md rounded-2xl border border-border/40 bg-card/90 backdrop-blur-xl p-6 shadow-2xl shadow-black/30 animate-in zoom-in-95 duration-200">
        {/* ─── Header ─── */}
        <div className="flex items-center justify-between mb-5">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center">
              <svg className="w-4 h-4 text-primary" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
              </svg>
            </div>
            <div>
              <h2 className="text-sm font-semibold">GitHub Authentication</h2>
              <p className="text-[11px] text-muted-foreground">Connect to enable PR, Issue & repo features</p>
            </div>
          </div>
          <button
            className="w-7 h-7 rounded-lg flex items-center justify-center text-muted-foreground hover:text-foreground hover:bg-accent/30 transition-colors"
            onClick={handleClose}
          >
            <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M18 6 6 18"/><path d="m6 6 12 12"/>
            </svg>
          </button>
        </div>

        {/* ─── Mode tabs ─── */}
        <div className="flex gap-1 mb-5 p-0.5 rounded-lg bg-muted/40 border border-border/20">
          <TabButton
            active={mode === "device"}
            onClick={() => { setMode("device"); resetAll(); }}
          >
            Sign in via Browser
          </TabButton>
          <TabButton
            active={mode === "pat"}
            onClick={() => { setMode("pat"); resetAll(); }}
          >
            Personal Access Token
          </TabButton>
        </div>

        {/* ─── Content ─── */}
        {mode === "device" ? <DeviceContent /> : <PATContent />}
      </div>
    </div>
  );

  /* ════════════════ Device Flow Content ════════════════ */

  function DeviceContent() {
    return (
      <div className="space-y-4">
        {deviceStep === "idle" && (
          /* ─── Initial state ─── */
          <div className="flex flex-col items-center gap-4 py-4">
            <div className="w-14 h-14 rounded-2xl bg-primary/8 flex items-center justify-center">
              <svg className="w-7 h-7 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
                <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
              </svg>
            </div>
            <div className="text-center space-y-1.5">
              <p className="text-sm font-medium">Connect to GitHub</p>
              <p className="text-xs text-muted-foreground max-w-xs">
                Authorize zgit via your browser. No tokens needed — just confirm one code on GitHub.
              </p>
            </div>
            <button
              onClick={handleStartDeviceFlow}
              className={buttonStyles("primary")}
            >
              <svg className="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
              </svg>
              Sign in via Browser
            </button>
          </div>
        )}

        {deviceStep === "starting" && (
          <LoadingState message="Starting device authorization..." />
        )}

        {(deviceStep === "awaiting" || deviceStep === "authenticating") && deviceCode && (
          /* ─── Code displayed, polling ─── */
          <div className="space-y-4">
            <div className="text-center space-y-2">
              <p className="text-xs text-muted-foreground">
                Enter the code below on GitHub
              </p>
              <div className="inline-flex items-center gap-1.5 px-5 py-3 rounded-xl bg-muted/50 border border-border/30 select-all">
                <span className="text-2xl font-mono font-bold tracking-[0.3em] text-foreground">
                  {deviceCode.user_code}
                </span>
              </div>
            </div>

            <div className="flex justify-center gap-2">
              <button
                onClick={() => {
                  navigator.clipboard.writeText(deviceCode.user_code);
                  setCopied(true);
                  setTimeout(() => setCopied(false), 2000);
                }}
                className={buttonStyles("secondary")}
              >
                {copied ? (
                  <>
                    <svg className="w-3.5 h-3.5 text-success" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <polyline points="20 6 9 17 4 12"/>
                    </svg>
                    Copied!
                  </>
                ) : (
                  <>
                    <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                    </svg>
                    Copy Code
                  </>
                )}
              </button>
              <button
                onClick={() => OpenURL(deviceCode.verification_uri || "https://github.com/login/device")}
                className={buttonStyles("primary")}
              >
                <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/>
                </svg>
                Open Browser
              </button>
            </div>

            <div className="flex flex-col items-center gap-2">
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <span className="w-3 h-3 border-2 border-primary border-t-transparent rounded-full animate-spin" />
                <span>
                  {deviceStep === "authenticating"
                    ? "Authorized! Saving token..."
                    : "Waiting for authorization..."}
                </span>
              </div>
            </div>

            <div className="flex justify-center">
              <button
                className="text-xs text-muted-foreground/60 hover:text-foreground transition-colors underline underline-offset-2"
                onClick={handleCancelDevicePoll}
              >
                Cancel
              </button>
            </div>
          </div>
        )}

        {deviceStep === "success" && (
          <div className="flex flex-col items-center gap-3 py-4">
            <div className="w-12 h-12 rounded-full bg-success/15 flex items-center justify-center">
              <svg className="w-6 h-6 text-success" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="20 6 9 17 4 12"/>
              </svg>
            </div>
            <p className="text-sm font-medium text-success">Connected to GitHub!</p>
          </div>
        )}

        {deviceStep === "error" && (
          <div className="space-y-4">
            <div className="flex items-start gap-2.5 p-3 rounded-lg bg-destructive/10 border border-destructive/20">
              <svg className="w-4 h-4 text-destructive shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
              </svg>
              <p className="text-xs text-destructive">{deviceError}</p>
            </div>
            <div className="flex justify-center gap-2">
              <button
                onClick={handleRetryDevice}
                className={buttonStyles("primary")}
              >
                <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/>
                </svg>
                Retry
              </button>
              <button
                onClick={handleClose}
                className={buttonStyles("secondary")}
              >
                Cancel
              </button>
            </div>
          </div>
        )}
      </div>
    );
  }

  /* ════════════════ PAT Content ════════════════ */

  function PATContent() {
    return (
      <form onSubmit={handlePatSubmit} className="space-y-4">
        <div className="text-xs text-muted-foreground space-y-1.5">
          <p>
            Enter a GitHub Personal Access Token to authenticate.
          </p>
          <p>
            Create one at{" "}
            <button
              type="button"
              onClick={() => OpenURL("https://github.com/settings/tokens")}
              className="text-primary underline underline-offset-2 hover:brightness-110 cursor-pointer"
            >
              github.com/settings/tokens
            </button>
            {" "}with <code className="text-[10px] bg-muted px-1 py-0.5 rounded font-mono">repo</code> and{" "}
            <code className="text-[10px] bg-muted px-1 py-0.5 rounded font-mono">read:user</code> scopes.
          </p>
        </div>

        <div className="relative">
          <input
            type={showToken ? "text" : "password"}
            value={token}
            onChange={(e) => setToken(e.target.value)}
            placeholder="ghp_xxxxxxxxxxxxxxxxxxxx"
            className="flex h-9 w-full rounded-lg border border-input bg-background/50 px-3 py-2 pr-14 text-xs font-mono ring-offset-background placeholder:text-muted-foreground/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring transition-shadow"
            autoFocus
            disabled={patLoading}
          />
          <button
            type="button"
            className="absolute right-2 top-1/2 -translate-y-1/2 text-[10px] font-medium text-muted-foreground hover:text-foreground px-1.5 py-0.5 rounded transition-colors"
            onClick={() => setShowToken(!showToken)}
            tabIndex={-1}
          >
            {showToken ? "Hide" : "Show"}
          </button>
        </div>

        {patError && (
          <div className="flex items-start gap-2 p-2.5 rounded-lg bg-destructive/10 border border-destructive/20">
            <svg className="w-3.5 h-3.5 text-destructive shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
            </svg>
            <p className="text-[11px] text-destructive">{patError}</p>
          </div>
        )}

        <div className="flex justify-end gap-2 pt-1">
          <button
            type="button"
            onClick={handleClose}
            disabled={patLoading}
            className={buttonStyles("secondary")}
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={!token.trim() || patLoading}
            className={cn(
              buttonStyles("primary"),
              (!token.trim() || patLoading) && "opacity-40 cursor-not-allowed"
            )}
          >
            {patLoading ? (
              <span className="flex items-center gap-1.5">
                <span className="w-3 h-3 border-2 border-current border-t-transparent rounded-full animate-spin" />
                Authenticating...
              </span>
            ) : (
              <span className="flex items-center gap-1.5">
                <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/><polyline points="10 17 15 12 10 7"/><line x1="15" y1="12" x2="3" y2="12"/>
                </svg>
                Sign In
              </span>
            )}
          </button>
        </div>
      </form>
    );
  }
}

/* ════════════════ Shared sub-components ════════════════ */

function TabButton({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "flex-1 text-xs font-medium px-3 py-1.5 rounded-md transition-all duration-150",
        active
          ? "bg-background text-foreground shadow-sm"
          : "text-muted-foreground hover:text-foreground"
      )}
    >
      {children}
    </button>
  );
}

function LoadingState({ message }: { message: string }) {
  return (
    <div className="flex flex-col items-center gap-3 py-6">
      <span className="w-5 h-5 border-2 border-primary border-t-transparent rounded-full animate-spin" />
      <p className="text-xs text-muted-foreground">{message}</p>
    </div>
  );
}

function buttonStyles(variant: "primary" | "secondary") {
  return cn(
    "inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-medium transition-all duration-150 press-scale",
    variant === "primary" && "bg-primary text-primary-foreground hover:brightness-110",
    variant === "secondary" && "border border-border/40 text-muted-foreground hover:text-foreground hover:bg-accent/20"
  );
}
