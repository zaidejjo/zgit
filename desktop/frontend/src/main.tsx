import React from "react";
import ReactDOM from "react-dom/client";
import { RouterProvider } from "@tanstack/react-router";
import router from "./router";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import "./index.css";

// Initialize theme from localStorage (must happen before first render)
const savedTheme = localStorage.getItem("zgit-theme");
if (savedTheme) {
  document.documentElement.dataset.theme = savedTheme;
} else {
  document.documentElement.dataset.theme = "dark";
}

/**
 * Persist the React root across HMR cycles.
 *
 * Without this, Vite re-executes main.tsx on hot-reload and tries to call
 * createRoot() on an element that already has a root — which throws.
 * By storing the root reference on `window`, we survive module re-execution
 * and can call `.render()` to update the tree with the fresh router instance.
 */
const ROOT_KEY = "__Z_GIT_ROOT";
const rootElement = document.getElementById("root")!;

function getOrCreateRoot(): ReactDOM.Root {
  const existing = (window as any)[ROOT_KEY] as ReactDOM.Root | undefined;
  if (existing) return existing;
  const root = ReactDOM.createRoot(rootElement);
  (window as any)[ROOT_KEY] = root;
  return root;
}

const root = getOrCreateRoot();

root.render(
  <React.StrictMode>
    <ErrorBoundary>
      <RouterProvider router={router} />
    </ErrorBoundary>
  </React.StrictMode>
);
