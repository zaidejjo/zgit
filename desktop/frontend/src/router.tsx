import { createRouter, createRootRoute, createRoute, createHashHistory } from "@tanstack/react-router";
import AppLayout from "./AppLayout";
import StatusPage from "./pages/StatusPage";
import LogPage from "./pages/LogPage";
import BranchesPage from "./pages/BranchesPage";
import PullRequestsPage from "./pages/PullRequestsPage";
import IssuesPage from "./pages/IssuesPage";
import ActionsPage from "./pages/ActionsPage";
import NotificationsPage from "./pages/NotificationsPage";
import RemotesPage from "./pages/RemotesPage";
import TagsPage from "./pages/TagsPage";
import WorktreesPage from "./pages/WorktreesPage";
import HistoryPage from "./pages/HistoryPage";
import ProjectsPage from "./pages/ProjectsPage";
import DashboardPage from "./pages/DashboardPage";
import SettingsPage from "./pages/SettingsPage";
import AIFocusPage from "./pages/AIFocusPage";

const rootRoute = createRootRoute({
  component: AppLayout,
});

const statusRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: StatusPage,
});

const logRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/log",
  component: LogPage,
});

const branchesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/branches",
  component: BranchesPage,
});

const pullRequestsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/pull-requests",
  component: PullRequestsPage,
});

const issuesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/issues",
  component: IssuesPage,
});

const actionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/actions",
  component: ActionsPage,
});

const notificationsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/notifications",
  component: NotificationsPage,
});

const remotesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/remotes",
  component: RemotesPage,
});

const tagsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/tags",
  component: TagsPage,
});

const worktreesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/worktrees",
  component: WorktreesPage,
});

const projectsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/projects",
  component: ProjectsPage,
});

const historyRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/history",
  component: HistoryPage,
});

const dashboardRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/dashboard",
  component: DashboardPage,
});

const settingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/settings",
  component: SettingsPage,
});

const aiRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/ai",
  component: AIFocusPage,
});

const routeTree = rootRoute.addChildren([
  statusRoute,
  logRoute,
  branchesRoute,
  pullRequestsRoute,
  issuesRoute,
  actionsRoute,
  notificationsRoute,
  remotesRoute,
  tagsRoute,
  worktreesRoute,
  historyRoute,
  dashboardRoute,
  projectsRoute,
  settingsRoute,
  aiRoute,
]);

// Use hash history — works in all URL schemes (file://, wails://, etc.)
const history = createHashHistory();

const router = createRouter({ routeTree, history });

export default router;
