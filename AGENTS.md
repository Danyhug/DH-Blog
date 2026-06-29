# Repository Guidelines

## Project Structure & Module Organization

DH-Blog is split into a Go backend and Vue frontend. Backend code lives in `blog-backend`: the entrypoint is `cmd/blog-backend/main.go`, with application layers under `internal/handler`, `internal/service`, `internal/repository`, `internal/model`, `internal/router`, and `internal/config`. Frontend code lives in `blog-front/src`, organized by `api`, `components`, `views`, `store`, `router`, `utils`, `types`, and `assets`. Integrated build and release files are in `blog-deploy`; generated binaries and runtime data should stay out of source changes unless intentionally updating deployment artifacts.

## Build, Test, and Development Commands

- `cd blog-front && bun run dev`: start Vite locally.
- `cd blog-front && bun run build`: build frontend assets into `dist`.
- `cd blog-front && bun run build:type-check`: run `vue-tsc`, then build; use before frontend PRs.
- `cd blog-backend && go run ./cmd/blog-backend`: run the backend.
- `cd blog-backend && go build ./cmd/blog-backend`: compile the backend.
- `cd blog-backend && go test ./...`: run all Go tests.
- `cd blog-deploy && ./build.sh`: build and embed the frontend, then produce binaries in `blog-deploy/build`.

## Coding Style & Naming Conventions

Format Go with `gofmt`; keep package names short and lowercase, and follow the existing `handler -> service -> repository -> model` flow. Frontend code uses Vue 3, TypeScript, Element Plus, Pinia, and Vite. Prefer `@/` imports, PascalCase Vue components, camelCase variables/functions, and kebab-case route paths or asset filenames. Keep generated files such as `auto-imports.d.ts` and `components.d.ts` in sync with Vite plugin output.

## Testing Guidelines

No frontend test runner is currently configured; validate frontend changes with `bun run build:type-check`. Backend tests should use Go's standard `testing` package, live beside the code as `*_test.go`, and run with `go test ./...`. For repository or handler changes, prefer table-driven tests covering success plus authorization or validation failures.

## Commit & Pull Request Guidelines

Git history follows Conventional Commit prefixes such as `feat:`, `fix:`, `refactor(scope):`, and `chore(deps):`; use a scope when it clarifies the area. Pull requests should describe the change, list validation commands, link related issues, and include screenshots for visible UI changes. Note config, database, or deployment impact explicitly.

## Security & Configuration Tips

Do not commit local secrets, runtime databases, uploaded files, or generated deployment data. Treat `.env`, `.env.production`, `data/`, `dhblog.db`, and build outputs as environment-specific. When changing authentication, file access, WebDAV, or backup behavior, call out migration and compatibility risks in the PR.
