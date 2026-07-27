# zgit — agent guidance

## Architecture

```
zgit/
├── cmd/zgit/          → CLI entrypoint (cobra)
├── internal/cli/      → CLI command implementations
├── pkg/core/          → shared engine (consumed by TUI + Desktop + CLI)
│   ├── engine.go      → Engine struct — composes GitAdapter, GitHubClient, Config, Cache
│   ├── git/           → GitAdapter interface + NativeExec (os/exec) implementation
│   ├── parser.go      → ParsePorcelainV2, ParseBranchInfo, ParseLogOutput
│   ├── github/        → REST + GraphQL GitHub clients
│   ├── ai/            → AI commit message generator (OpenAI, Anthropic, etc.)
│   ├── config/        → Viper YAML at ~/.config/zgit/config.yaml
│   ├── models/        → domain types (shared across all layers)
│   └── cache/         → TTL-aware LRU wrapper
├── pkg/tui/           → Bubble Tea TUI
└── desktop/           → Wails v2 desktop app (separate Go module, replace directive)
    ├── app.go         → ~60 exported methods, all become JS-callable via Bind
    ├── frontend/      → React 18 + Vite + TypeScript + shadcn/ui + Zustand
    └── build/         → embedded via //go:embed frontend/dist
```

## Build & verify

```bash
# Go (CLI + TUI + core)
make build        # go build -o zgit ./cmd/zgit/
make test         # go test ./... -count=1 -timeout=60s
make lint         # golangci-lint run ./... | skips if not installed
make vet          # go vet ./...
make fmt          # go fmt ./...
make dev          # fmt → vet → test → build

# Desktop frontend only (Wails-less)
cd desktop/frontend && npm run build   # tsc && vite build

# Full desktop binary (requires Wails CLI + webkit2gtk-4.0)
cd desktop && wails build
```

## Critical gotchas

### Git exit code 128 = empty repo / unborn HEAD
- `git branch --format=`, `rev-parse --abbrev-ref HEAD` on empty repo → exit 128
- `NativeExec` must catch in both Go backend and frontend store
- Pattern: `if asGitErr(err, &gitErr) && gitErr.ExitCode == 128 { return defaultValue, nil }`
- Store: `fetchLog`, `fetchBranches` catch 128 → set empty array + skip error toast

### Go nil slice → JSON `null` → frontend crash
- `ParseRemotes()` must use `make([]*models.Remote, 0)` not `var remotes []*models.Remote`
- Frontend guards: `remotes || []`, `branches || []` in store/page

### Store state on repo switch
- `openRepo` resets all stale state: status, log, branches, diff, currentBranch, tags, reflog, undoDescription, conflictFiles, mergeEditorOpen, mergeEditorFile, mergeConflictDetail, rebaseMode, rebaseCommits, stashes, remotes, error — before `refreshAll()`

### Wails JS bridge
- All exported `App` methods become `window.go.main.App.<MethodName>(args)`
- Store accesses via `getBackend()` helper: `window.go?.main?.App || window.Go?.main?.App`
- Model classes auto-generated in `wailsjs/go/models.ts` — do NOT edit
- TS declarations in `wailsjs/go/main/App.d.ts` — do NOT edit
- Runtime events: `EventsOn("fs:status-changed", ...)`, `EventsOn("repo:switched", ...)`

### Frontend HMR
- React root persisted on `window.__Z_GIT_ROOT` to survive Vite module re-execution
- `main.tsx` implements `getOrCreateRoot()` — do not break this pattern

## Code conventions

### Go
- `GitAdapter` interface in `adapter.go` (40 methods), `NativeExec` in `native.go` (~1281 lines)
- All `NativeExec` methods safe for concurrent use (`sync.RWMutex`)
- Git errors wrapped with `GitError{ExitCode, Stderr}` — use `asGitErr()` to unwrap
- Options structs for all operations: `LogOptions`, `DiffOptions`, `PullOptions`, `PushOptions`, etc.

### TypeScript / React
- **Snake_case** interfaces matching Go structs (Wails JSON serialization uses snake_case)
- Single Zustand store at `src/store/app.ts` (~1759 lines)
- `@/*` path alias maps to `./src/*`
- shadcn/ui with Zinc base color, CSS variables, dark mode via `class`
- Dark mode forced on at startup (`main.tsx:9`)
- All navigation via `@tanstack/react-router` with hash history
- No frontend test framework or test files

### Styling
- Tailwind CSS v3 with `class-variance-authority` + `clsx` + `tailwind-merge` (`cn()` helper)
- Custom CSS vars: `--git-added` (green), `--git-modified` (yellow), `--git-deleted` (red), `--git-untracked`, `--pr-open`, `--pr-closed`, `--pr-merged`

### Commits
- Conventional Commits: `feat:`, `fix:`, `refactor:`, `chore:`, `docs:`, `style:`, `test:`

## Testing
- 3 Go test files: `parser_test.go` (table-driven), `rest_test.go` (httptest.Server), `cache_test.go`
- No frontend tests, no E2E, no integration tests
- No CI/CD

## Unusual dependencies
- `@dnd-kit/core`, `@dnd-kit/sortable`, `@dnd-kit/utilities` — drag-and-drop commit graph in LogPage
- `zustand` v5 — state management (single store)
- `lucide-react` — icons
- `charmbracelet/bubbletea` — TUI
- `spf13/cobra` — CLI
- `shurcooL/githubv4` — GitHub GraphQL API
- `fsnotify/fsnotify` — file watcher (desktop only)
