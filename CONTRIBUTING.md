# Contributing to Zgit

## Setup

```bash
git clone https://github.com/zaidejjo/zgit.git
cd zgit
make dev    # fmt → vet → test → build

# Desktop frontend
cd desktop/frontend && bun install && bun run build
```

Prerequisites: Go 1.26, git, bun. Desktop builds additionally need the
Wails CLI + webkit2gtk.

## Workflow

- **One branch per feature**: `feat/<name>` branched from `dev`.
- Open a PR against `dev`. CI (`ci.yml`: vet, gofmt, tests, frontend build)
  must be green — merges require it.
- Keep PRs focused; one feature, one PR.
- `dev` is the integration branch. Releases are cut from `main` via version tags.

## Commits

Conventional Commits: `feat:`, `fix:`, `refactor:`, `chore:`, `docs:`, `style:`, `test:`.
Release notes and changelogs are generated from these, so keep them clean.

## Tests

- Unit tests live next to source: `pkg/.../*_test.go`.
- Integration tests (real git repos in temp dirs) live in `tests/`.
- New backend features need both. Run everything with `go test ./...`.
- CLI changes: smoke-test the built binary against a scratch repo.
- Frontend changes: `bun run build` (tsc + vite) must pass.

## Architecture

See [AGENTS.md](AGENTS.md) for the authoritative map: `pkg/core` engine,
`GitAdapter` + `NativeExec`, Wails bindings in `desktop/app.go`, the single
Zustand store, and the gotchas (exit 128, nil slices, Wails bridge).

## What makes a good PR

- Follows existing patterns (check `adapter.go` + `native.go` + `app.go` + store before adding).
- No placeholders or TODOs.
- Handles errors; surfaces them to the user instead of swallowing.
- Updates DOCS.md when behavior or installation changes.
