# Repository Guidelines

## Project Structure & Module Organization

DH-Blog contains a Go backend and a Vue frontend. Backend code lives in `blog-backend`; the entry point is `cmd/blog-backend/main.go`. Business features are organized as vertical modules under `internal/modules` (for example, `article`, `files`, and `user`), while shared infrastructure lives in `internal/app`, `internal/database`, `internal/middleware`, `internal/platform`, and `internal/router`. Backend tests sit beside implementation files as `*_test.go`.

Frontend code lives in `blog-front/src`, grouped into `api`, `components`, `views`, `store`, `router`, `utils`, `types`, and `assets`. Deployment scripts and integrated build outputs belong in `blog-deploy`. Do not commit generated binaries, runtime databases, uploads, or local deployment data.

## Build, Test, and Development Commands

- `cd blog-front && bun run dev`: start the Vite development server.
- `cd blog-front && bun run build:type-check`: run `vue-tsc`, then build production assets.
- `cd blog-front && bun run preview`: preview the production frontend build.
- `cd blog-backend && go run ./cmd/blog-backend`: run the backend locally.
- `cd blog-backend && go test ./...`: run all Go tests.
- `cd blog-backend && go build ./cmd/blog-backend`: compile the backend executable.
- `cd blog-deploy && ./build.sh`: build the integrated release artifacts.

## Coding Style & Naming Conventions

Format Go with `gofmt`; keep package names short and lowercase. Add business behavior to the appropriate vertical module and preserve the existing handler/service/repository boundaries where present. For Vue and TypeScript, use two-space indentation, PascalCase component names, camelCase variables and functions, `@/` imports, and kebab-case route paths or asset names. Keep generated declarations such as `auto-imports.d.ts` synchronized with the Vite plugins.

## Testing Guidelines

Use Go's standard `testing` package and table-driven tests where practical. Cover success paths plus validation, authorization, rollback, or storage failures relevant to the change. No frontend test runner is configured; validate frontend work with `bun run build:type-check` and manually inspect visible changes.

## Commit & Pull Request Guidelines

Follow Conventional Commit prefixes used in history, such as `feat:`, `fix:`, `refactor(scope):`, and `chore(deps):`. Pull requests should explain the change, list validation commands, link related issues, and include screenshots for UI changes. Explicitly note configuration, database, migration, security, or deployment impact.

## Security & Configuration Tips

Never commit secrets, `.env` files, `data/`, `dhblog.db`, uploaded content, or build outputs. Review path validation and compatibility carefully when changing authentication, file access, WebDAV, backup, or storage behavior.
