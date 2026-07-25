import { createRouter, createRootRoute, createRoute } from "@tanstack/react-router";
import AppLayout from "./AppLayout";
import StatusPage from "./pages/StatusPage";
import LogPage from "./pages/LogPage";
import BranchesPage from "./pages/BranchesPage";
import PullRequestsPage from "./pages/PullRequestsPage";
import IssuesPage from "./pages/IssuesPage";

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

const routeTree = rootRoute.addChildren([
  statusRoute,
  logRoute,
  branchesRoute,
  pullRequestsRoute,
  issuesRoute,
]);

const router = createRouter({ routeTree });

export default router;
