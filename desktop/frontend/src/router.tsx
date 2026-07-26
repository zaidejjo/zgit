import { createRouter, createRootRoute, createRoute, createHashHistory } from "@tanstack/react-router";
import AppLayout from "./AppLayout";
import StatusPage from "./pages/StatusPage";
import LogPage from "./pages/LogPage";
import BranchesPage from "./pages/BranchesPage";
import PullRequestsPage from "./pages/PullRequestsPage";
import IssuesPage from "./pages/IssuesPage";
import ActionsPage from "./pages/ActionsPage";

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

const routeTree = rootRoute.addChildren([
  statusRoute,
  logRoute,
  branchesRoute,
  pullRequestsRoute,
  issuesRoute,
  actionsRoute,
]);

// Use hash history — works in all URL schemes (file://, wails://, etc.)
const history = createHashHistory();

const router = createRouter({ routeTree, history });

export default router;
