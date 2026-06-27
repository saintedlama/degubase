# E2E Tests

Playwright tests organized by feature domain.

## Structure

```
e2e/
├── fixtures.ts          # Shared helpers: createWorkspace, createTable, seedRow, cleanup
├── smoke.spec.ts        # Full happy-path (create workspace → table → column → row → edit → delete)
├── mobile.spec.ts       # Mobile UX regressions (#50, #51, #52, #79)
├── workspace/
│   ├── codes.spec.ts    # Immutable 4-char workspace/table codes (UI creation flows)
│   ├── icon.spec.ts     # Table icon picker (#99)
│   └── modals.spec.ts   # Edit/delete modal flows (#90)
├── schema/
│   ├── column-code.spec.ts  # Column code feature (API + UI)
│   ├── checklist.spec.ts    # Checklist column type (flapout, items, progress bar)
│   ├── markdown.spec.ts     # Markdown editor fast-typing fix (#98)
│   ├── row-link.spec.ts     # Row-link column (picker, clear, blocked deletes, display column)
│   └── views.spec.ts        # View config persistence (matrix, kanban card fields)
├── records/
│   ├── annotations.spec.ts  # Activity panel annotations (#96)
│   ├── bulk.spec.ts         # Bulk edit dialog and API (#71)
│   ├── csv.spec.ts          # CSV import/export (#67)
│   └── record-detail.spec.ts # Record terminology (#94, #95) + loading indicators (#75)
└── live/
    └── sse.spec.ts          # Kanban SSE live-update (#97)
```

## What gets e2e-tested

E2e tests cover **user-observable behaviour** that unit/component tests cannot
reliably verify:

- **Navigation flows** — URLs update correctly, back-button works
- **Persistence** — data survives a full page reload (real network round-trip)
- **Cross-component wiring** — e.g. a cell edit in the table view reflects in
  the record detail view
- **API/UI contract** — API responses drive what the UI renders (label, count,
  state)
- **Regressions** — specific bugs that were fixed and must not recur (every
  `#NN` comment in a test names the issue)

E2e tests do **not** duplicate unit-level logic (field validation rules,
computed values, store mutations). Those belong in Vitest.

## Fixtures (`fixtures.ts`)

| Export | Purpose |
|---|---|
| `createWorkspace(request, name?)` | POST a workspace, return its code |
| `createTable(request, wsCode, name, cols?)` | POST table + columns, return `{ code, viewCode, url }` |
| `seedRow(request, wsCode, tableCode, data?)` | POST a row, return its id |
| `cleanup(request, wsCode)` | DELETE the workspace (cascades everything) |
| `test` (extended) | Provides a `wsCode` fixture — one isolated workspace per test, auto-cleaned |

## Serial vs parallel

Tests within a `describe` block run **in parallel by default**. A block uses
`test.describe.configure({ mode: 'serial' })` only when tests genuinely chain
state (e.g. checklist tests that add items, check them, then verify the count).

Use `beforeAll` for API-only setup. Only keep a setup `test()` when the UI
creation flow itself is what is being tested (e.g. `workspace/codes.spec.ts`).

## Running

```sh
# All tests (starts servers automatically via webServer config)
pnpm exec playwright test

# A single domain
pnpm exec playwright test e2e/schema/

# One file
pnpm exec playwright test e2e/records/bulk.spec.ts

# With UI explorer
pnpm exec playwright test --ui
```
