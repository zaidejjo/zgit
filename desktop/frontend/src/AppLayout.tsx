import { useEffect } from "react";
import { Outlet, useNavigate, useLocation } from "@tanstack/react-router";
import { useAppStore } from "@/store/app";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const tabs = [
  { id: "status", label: "Status", icon: "◉" },
  { id: "log", label: "Log", icon: "◈" },
  { id: "branches", label: "Branches", icon: "⑂" },
  { id: "pull-requests", label: "PRs", icon: "◆" },
  { id: "issues", label: "Issues", icon: "●" },
] as const;

export default function AppLayout() {
  const { activeTab, setActiveTab, error, setError, darkMode, toggleDarkMode, fetchStatus, checkGitHubAuth } =
    useAppStore();
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    document.documentElement.classList.toggle("dark", darkMode);
  }, [darkMode]);

  useEffect(() => {
    fetchStatus();
    checkGitHubAuth();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleTabChange = (tabId: string) => {
    setActiveTab(tabId);
    navigate({ to: `/${tabId === "status" ? "" : tabId}` });
  };

  // Auto-dismiss error after 5s
  useEffect(() => {
    if (error) {
      const timer = setTimeout(() => setError(null), 5000);
      return () => clearTimeout(timer);
    }
  }, [error, setError]);

  const currentTab = tabs.find((t) =>
    location.pathname === "/" ? t.id === "status" : location.pathname === `/${t.id}`
  )?.id || "status";

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b">
        <div className="flex items-center justify-between px-4 h-12">
          <div className="flex items-center gap-1">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => handleTabChange(tab.id)}
                className={cn(
                  "px-3 py-1.5 text-sm font-medium rounded-md transition-colors",
                  currentTab === tab.id
                    ? "bg-primary text-primary-foreground"
                    : "text-muted-foreground hover:text-foreground hover:bg-accent"
                )}
              >
                <span className="mr-1.5">{tab.icon}</span>
                {tab.label}
              </button>
            ))}
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2"
              onClick={toggleDarkMode}
            >
              {darkMode ? "☀️" : "🌙"}
            </Button>
          </div>
        </div>
      </header>

      {/* Error toast */}
      {error && (
        <div className="fixed top-4 right-4 z-50 max-w-md bg-destructive text-destructive-foreground px-4 py-3 rounded-lg shadow-lg text-sm">
          {error}
          <button
            className="ml-3 text-destructive-foreground/70 hover:text-destructive-foreground"
            onClick={() => setError(null)}
          >
            ✕
          </button>
        </div>
      )}

      {/* Main content */}
      <main className="p-4">
        <Outlet />
      </main>
    </div>
  );
}
