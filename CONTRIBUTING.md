# Contributing to Degubase

Thank you for your interest in contributing to Degubase! Degubase is built as the "anti-platform" — a personal, lightweight, single-binary database with a reactive web UI and automation engine. We welcome contributions that help improve performance, add useful features, fix bugs, and refine documentation.

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

---

## Table of Contents

- [How Can I Contribute?](#how-can-i-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Features & Enhancements](#suggesting-features--enhancements)
  - [Submitting Pull Requests](#submitting-pull-requests)
- [Development Environment Setup](#development-environment-setup)
  - [Prerequisites](#prerequisites)
  - [Initial Setup](#initial-setup)
- [Development Workflow](#development-workflow)
  - [Starting Development Servers](#starting-development-servers)
  - [Testing](#testing)
  - [Linting & Formatting](#linting--formatting)
  - [Updating OpenAPI Docs](#updating-openapi-docs)
  - [Seeding Data](#seeding-data)
- [Codebase Architecture](#codebase-architecture)
- [Pull Request Guidelines](#pull-request-guidelines)
- [License Notice](#license-notice)

---

## How Can I Contribute?

### Reporting Bugs

Before creating a bug report, please search existing GitHub issues to verify the problem hasn't already been reported.

If you find a new bug, please file an issue on GitHub and include:
- A clear, descriptive title.
- Steps to reproduce the issue.
- Expected vs. actual behavior.
- Environment details (OS, Go version, Node/browser version).
- Relevant log output or error stack traces.

### Suggesting Features & Enhancements

We are eager to hear ideas for improving Degubase! When proposing a new feature:
- Check existing issues and discussions to see if it has already been suggested.
- Open an issue on GitHub describing your proposal.
- Explain the motivation, use case, and how the user or agent would interact with the feature.
- For substantial architectural changes, consider submitting an RFC issue before opening a pull request.

### Submitting Pull Requests

Small fixes, typos, documentation improvements, and bug fixes are welcome directly as Pull Requests. For larger features or changes to core abstractions, opening an issue first is recommended to discuss the design.

---

## Development Environment Setup

### Prerequisites

Make sure the following tools are installed:
- [Go](https://go.dev) 1.27 or newer
- [Node.js](https://nodejs.org) 22 or newer
- [pnpm](https://pnpm.io) 11+
- [air](https://github.com/air-verse/air) (for Go live reloading in development)
- [swag](https://github.com/swaggo/swag) v1.16+ (for OpenAPI doc regeneration)

Install Go tools:
```bash
go install github.com/air-verse/air@latest
go install github.com/swaggo/swag/cmd/swag@v1.16.6
```

### Initial Setup

1. Fork and clone the repository:
   ```bash
   git clone https://github.com/<your-username>/degubase.git
   cd degubase
   ```
2. Install frontend dependencies:
   ```bash
   cd ui && pnpm install && cd ..
   ```
3. Install E2E test dependencies (optional, for browser tests):
   ```bash
   cd e2e && pnpm install && pnpm exec playwright install chromium && cd ..
   ```

---

## Development Workflow

### Starting Development Servers

To start the full development environment with live reloading:

```bash
make dev
```

Alternatively, you can run the backend and frontend separately:

- **Backend** (API server on port 8080):
  ```bash
  air
  # or directly:
  go run ./cmd/server
  ```
- **Frontend** (Vite dev server with HMR, proxying to backend):
  ```bash
  cd ui
  pnpm dev
  ```

### Testing

Always run tests before submitting a pull request:

```bash
# Run all Go unit and integration tests
make test

# Run Playwright E2E smoke tests
make e2e

# Run Playwright interactive UI test runner
make e2e-ui

# Generate test coverage report
make coverage
```

### Linting & Formatting

Degubase enforces strict formatting and linting in CI:

```bash
# Auto-format all Go source files
make fmt

# Run all linters (gofmt, go vet, staticcheck, deadcode, go mod tidy)
make lint
```

Ensure frontend code conforms to formatting conventions:
```bash
cd ui && pnpm lint
```

### Updating OpenAPI Docs

Degubase generates OpenAPI 3.0 documentation from declarative Go comment annotations (`swaggo/swag`). If you modify, add, or delete any HTTP endpoints in `internal/api/`, you must regenerate the documentation:

```bash
make generate
```

Verify that any changed files in `docs/` are committed with your PR.

### Seeding Data

When testing views, column types, or performance:

```bash
# Seed 6 curated demo workspaces (CRM, Bookmarks, Bug Tracker, etc.)
make seed-demo

# Seed large dataset for performance benchmarks (100 tables × 20 columns × 1000 rows)
make seed-perf

# Wipe and recreate local database
make db-reset
```

---

## Codebase Architecture

```
degubase/
├── cmd/
│   ├── server/             # Main application entry point
│   ├── seed_demo/          # Demo database seeding CLI
│   └── seed_perf/          # Performance benchmark database seeder
├── internal/
│   ├── api/                # HTTP routing (Chi), middleware, auth, OpenAPI handlers
│   ├── automation/         # Lua scripting engine (gopher-lua), triggers, script CRUD
│   ├── identity/           # Users, JWT auth, bcrypt passwords, workspace tokens
│   ├── infrastructure/     # Events, SQLite storage helpers, HTTP utils
│   ├── models/             # Domain types, errors, shared models
│   ├── records/            # Row CRUD, cell value storage, CSV import/export, file storage, graph traversal, SSE
│   └── schema/             # Workspace, Table, Column, and View definitions
├── ui/                     # Vue 3 frontend (Vite, Pinia/reactive stores, Vue Router, Tailwind CSS)
├── e2e/                    # Playwright end-to-end integration tests
├── docs/                   # Auto-generated OpenAPI / Swagger specs
└── Makefile                # Project automation tasks
```

### Adding New Column Types or Views

- **Column Types**: Register new types in `internal/schema` and `internal/records` (backend validation and storage), then implement corresponding display and edit components in `ui/src/components/`.
- **Views**: Define view schemas in `internal/schema` and create view components in `ui/src/views/`.

---

## Pull Request Guidelines

1. **Branch Naming**: Use descriptive branch names like `feat/rating-column-color`, `fix/csv-export-nulls`, or `docs/update-quickstart`.
2. **Atomic Commits**: Make small, logical commits with clear messages. We recommend [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat: add symbol column type`
   - `fix: handle cyclic row links gracefully`
   - `docs: clarify Lua event triggers`
   - `test: add e2e test for Kanban column reordering`
3. **Keep it green**: Ensure `make lint` and `make test` pass locally before pushing.
4. **Documentation**: If your PR introduces user-facing changes, update the relevant sections of `README.md` or API annotations.

---

## License Notice

Degubase is licensed under the [GNU General Public License v3.0 (GPLv3)](LICENSE). By contributing to Degubase, you agree that your contributions will be licensed under the terms of the GNU General Public License v3.0.
