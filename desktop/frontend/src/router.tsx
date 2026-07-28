import { createRouter, createRootRoute, createRoute, createHashHistory } from "@tanstack/react-router";
import AppLayout from "./AppLayout";
import StatusPage from "./pages/StatusPage";
import LogPage from "./pages/LogPage";
import BranchesPage from "./pages/BranchesPage";
import PullRequestsPage from "./pages/PullRequestsPage";
import IssuesPage from "./pages/IssuesPage";
import ActionsPage from "./pages/ActionsPage";
import RemotesPage from "./pages/RemotesPage";
import TagsPage from "./pages/TagsPage";
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
  remotesRoute,
  tagsRoute,
  settingsRoute,
  aiRoute,
]);

// Use hash history — works in all URL schemes (file://, wails://, etc.)
const history = createHashHistory();

const router = createRouter({ routeTree, history });

export default router;
