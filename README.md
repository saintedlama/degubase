# Degubase

<p align="center">
  <img src="ui/public/assets/logo.svg" width="80" height="80" alt="Degubase Logo" />
</p>

<p align="center">
  <strong>The database UI for personal tooling, side projects, and small teams.</strong><br />
  A single binary that spins up in seconds — no accounts, no paywalls, no external database servers.<br />
  Just you, your data, and an instant Model Context Protocol (MCP) server for your AI agents.
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-GPLv3-blue.svg" alt="License: GPL v3" /></a>
  <a href="https://github.com/saintedlama/degubase/releases"><img src="https://img.shields.io/github/v/release/saintedlama/degubase?color=emerald" alt="Release" /></a>
  <a href="https://github.com/saintedlama/degubase/pkgs/container/degubase"><img src="https://img.shields.io/badge/docker-ghcr.io-blue?logo=docker" alt="Docker Container" /></a>
  <a href="https://modelcontextprotocol.io"><img src="https://img.shields.io/badge/MCP-Compatible-purple" alt="Model Context Protocol" /></a>
</p>

<p align="center">
  <a href="#-quickstart">⚡ Quickstart</a> •
  <a href="#-key-features">✨ Key Features</a> •
  <a href="#-five-interactive-views">📊 Views</a> •
  <a href="#-native-mcp-server-for-ai-agents">🤖 MCP for AI</a> •
  <a href="#-ready-made-templates">📦 Templates</a> •
  <a href="#-showcase">🖼️ Showcase</a> •
  <a href="#-configuration--api">⚙️ Config & API</a>
</p>

---

![Degubase Home — Workspaces](screenshots/01-home.png)

---

## Why Degubase?

We all have data to wrangle. Spreadsheets often feel too loose, full-blown database engines require setup and hosting, and SaaS platforms demand your email, credit card, and privacy.

**Degubase is the anti-platform:** a single compiled binary that gives you a complete relational database with a clean, responsive web UI sitting directly on your machine. 

- **100% Local & Self-Contained** — Backed by SQLite. Everything lives in one file or data directory. Zero Redis, Postgres, or container sprawl.
- **Built for Humans & AI Agents** — Human-friendly web views with inline editing, plus an instant Model Context Protocol (MCP) server so Claude, Cursor, or your custom agents can read and write data reliably.
- **Your Data, Your Rules** — Attach Lua scripts that react to row changes, link records across tables, and import/export CSV whenever you want.

---

## ⚡ Quickstart

### Option 1: Docker (Fastest)

Run Degubase with a single command. Data persists automatically in a named volume:

```bash
docker run -d \
  --name degubase \
  -p 8080:8080 \
  -v degubase-data:/data \
  -e DEGUBASE_HOST=0.0.0.0 \
  ghcr.io/saintedlama/degubase:latest
```

Open **`http://localhost:8080`** in your browser.

<details>
<summary><b>Using Docker Compose? Click for <code>compose.yml</code></b></summary>

```yaml
services:
  degubase:
    image: ghcr.io/saintedlama/degubase:latest
    container_name: degubase
    ports:
      - "8080:8080"
    environment:
      DEGUBASE_HOST: "0.0.0.0"
    volumes:
      - data:/data
      - config:/config
    restart: unless-stopped

volumes:
  data:
  config:
```

Run with:
```bash
docker compose up -d
```
</details>

---

### Option 2: Run from Source (Go)

```bash
git clone https://github.com/saintedlama/degubase.git
cd degubase
go run ./cmd/server
```

A SQLite database (`degubase.db`) is automatically initialized on first run.

> 💡 **Tip:** Want to explore Degubase with sample data preloaded? Run `make seed-demo` (or `go run ./cmd/seed_demo`) to instantly populate demo workspaces for Task Tracking, CRM, Bug Tracking, Content Planning, and Recipes!

---

## ✨ Key Features

- **Single Binary, Zero Dependencies** — Go backend embeds the complete Vue 3 frontend bundle. Run one executable and you're up.
- **20+ Rich Column Types** — Designed for real-world information:
  `text`, `long-text`, `markdown`, `email`, `url`, `number`, `currency`, `percent`, `rating`, `date`, `datetime`, `checkbox`, `single-select`, `multi-select`, `checklist`, `file`, `image` (with auto-thumbnails), `emoji`, `symbol`, `row-link`, `created-at`, and `updated-at`.
- **Linked Records & Hierarchies** — Connect rows across tables (*assignee ➔ user*, *invoice ➔ client*) or create parent/child trees within the same table (*subtask ➔ task*, *nested pages*). Graph traversal API included with circular reference protection.
- **Event-Driven Lua Automations** — Write lightweight Lua scripts triggered on row `create`, `update`, or `delete`. Compute derived fields, validate inputs, enforce business logic, or make outbound HTTP requests.
- **Real-Time UI Sync (SSE)** — Server-Sent Events stream database changes live. Boards, grids, and timelines refresh automatically across open browser tabs.
- **Row History & Audit Trail** — Every edit is tracked with revision IDs and optional user annotations.
- **Workspace Security & API Tokens** — Optional JWT login, bcrypt password hashing, and scoped SHA-256 bearer tokens for integrations. Or set `disable_auth: true` for frictionless single-user local use.

---

## 📊 Five Interactive Views

Switch perspectives on the same table with one click:

| View | Best For | Features |
|---|---|---|
| **Tabular** | Data entry & spreadsheet workflows | Inline cell editing, column resizing, bulk operations, row context menus. |
| **Kanban** | Workflow & pipeline tracking | Group by any single-select column, drag-and-drop card movement. |
| **Timeline** | Editorial calendars & project roadmaps | Swimlanes grouped by channel/owner, date-range scheduling, drag to shift dates. |
| **Matrix** | Risk registers & priority grids | 2D cross-referencing over X and Y select axes (e.g. Likelihood vs Impact). |
| **Card / Gallery** | Portfolios, assets & recipe collections | Visual card layouts, image cover previews, configurable visible fields. |

---

## 🤖 Native MCP Server for AI Agents

Every Degubase workspace exposes a native **Model Context Protocol (MCP)** server over Server-Sent Events (SSE). This turns Degubase into a persistent, structured memory and tooling layer for AI assistants like Claude Desktop, Cursor, and custom agents.

### Quick Setup (Claude Desktop)

1. In Degubase, open your workspace settings, enable the **MCP Server** toggle, and generate an **API Token**.
2. Add the server to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "degubase": {
      "url": "http://localhost:8080/api/workspaces/<WORKSPACE_CODE>/mcp/sse",
      "headers": {
        "Authorization": "Bearer <YOUR_WORKSPACE_TOKEN>"
      }
    }
  }
}
```

### Why Agents Love Degubase:
- **Semantic Context Awareness**: Workspaces and tables carry descriptive `context` fields that explain schema semantics to LLMs before they write a query.
- **20+ Dedicated Tools**: AI agents can query rows, filter data, inspect linked relations, create tables, update columns, inspect row histories, and run automations programmatically.
- **OpenAPI 3.0 Documentation**: Auto-generated live specification at `/api/openapi.json` for custom LLM tooling and API clients.

---

## 📦 Ready-Made Templates

Degubase includes 16 pre-configured templates that spin up instantly with tailored column schemas and preset default views:

- 🚀 **Product & Engineering**:
  - *Product Roadmap* (Now/Next/Later with Kanban & Timeline)
  - *Bug Tracker* (Markdown reproduction steps, screenshot uploads, triage board & severity matrix)
  - *Releases & Changelog* (Semantic version tracking, breaking changes, timeline view)
  - *Incident Log & Post-Mortem* (Outage timestamps, root causes, checklist action items)
  - *Goals & Bets* (Quarterly bets, key progress metrics, and progress percentages)
  - *Feature Requests* (User feedback, pain points, demand matrix)
- 💼 **Sales & Operations**:
  - *CRM & Contacts* (Contact records, interaction dates, status tracking)
  - *Sales Pipeline* (Deal stages, currency values, target close dates)
  - *Client Projects & Billing* (Client deliverables, contract values, invoice status)
  - *Launch Checklist* (Cross-functional go-live sanity checks)
- 🧠 **Strategy & Personal**:
  - *Risk Register* (Pre-built Likelihood × Impact 2D Matrix view)
  - *OKRs (Objectives & Key Results)* (Measurable targets and initiatives)
  - *Decision Journal* (Context, alternatives considered, expected vs. actual outcomes)
  - *Recipe Book*, *Reading List*, *Subscriptions & Renewals*
- 🤖 **Coding Agents**:
  - *Agent Orchestration*, *Evidence Trail*, *Task Memory Graph*

---

## 🖼️ Showcase

### Workflows & Views

| Kanban Board View | Timeline Schedule View |
|---|---|
| ![Task Tracker board — tasks kanban](screenshots/03-task-tracker-board.png) | ![Content Calendar timeline](screenshots/07-content-calendar-timeline.png) |
| *Drag-and-drop workflow tracking grouped by column.* | *Visual swimlane scheduling across date ranges.* |

| Tabular Spreadsheet Grid | Card / Gallery View |
|---|---|
| ![Bug Tracker grid](screenshots/04-bug-tracker-grid.png) | ![Recipe Book gallery](screenshots/06-recipe-book-gallery.png) |
| *Fast spreadsheet editing with rich types and filters.* | *Cover images, ratings, badges, and quick glance fields.* |

### Pipelines & Details

| Sales & Deal Pipeline | Detailed Record View |
|---|---|
| ![CRM pipeline](screenshots/05-crm-pipeline.png) | ![Row detail view](screenshots/08-row-detail.png) |
| *Deal values, company tags, and stage pipelines.* | *Markdown rendering, rich attachments, and row actions.* |

---

## ⚙️ Configuration & API

<details>
<summary><b>🔧 Configuration Options (YAML & Environment Variables)</b></summary>
<br />

Configure via `degubase.yaml` in your working directory or standard environment variables:

| YAML key | Environment variable | Default | Description |
|---|---|---|---|
| `port` | `DEGUBASE_PORT` | `8080` | HTTP listen port |
| `host` | `DEGUBASE_HOST` | `127.0.0.1` | Network interface to bind (`0.0.0.0` for containers/LAN) |
| `data_dir` | `DEGUBASE_DATA_DIR` | `data` | Directory for SQLite database and uploaded files |
| `jwt_secret` | `DEGUBASE_JWT_SECRET` | *(auto)* | Secret used to sign authentication JWT tokens |
| `disable_auth` | `DEGUBASE_DISABLE_AUTH` | `false` | Disables login requirement (auto-authenticates as admin) |
| `automation.http.allowed_hosts` | `DEGUBASE_AUTOMATION_HTTP_ALLOWED_HOSTS` | `[]` | Hosts or `host:port` allowed in Lua `http.get`/`post` bypassing SSRF checks |

</details>

<details>
<summary><b>🔌 REST API Quick Reference</b></summary>
<br />

Every table and record is accessible via REST with standard JSON payloads:

```bash
# 1. Create a workspace
curl -X POST http://localhost:8080/api/workspaces \
  -H "Content-Type: application/json" \
  -d '{"name":"Product Ops","context":"Product development and bug tracking"}'

# 2. Create a table
curl -X POST http://localhost:8080/api/workspaces/PRODUCT_OPS/tables \
  -H "Content-Type: application/json" \
  -d '{"name":"Tasks","context":"Sprint engineering tasks"}'

# 3. Add a column
curl -X POST http://localhost:8080/api/workspaces/PRODUCT_OPS/tables/TASKS/columns \
  -H "Content-Type: application/json" \
  -d '{"name":"Status","type":"single-select","options":{"choices":[["Todo","#808080"],["Done","#22c55e"]]}}'

# 4. Insert a record
curl -X POST http://localhost:8080/api/workspaces/PRODUCT_OPS/tables/TASKS/rows \
  -H "Content-Type: application/json" \
  -d '{"data":{"title":"Implement OAuth","Status":"Todo"}}'
```

Full interactive documentation is available at `http://localhost:8080/api/openapi.json`.
</details>

<details>
<summary><b>🛠️ Local Development & Makefile Guide</b></summary>
<br />

Degubase is built with **Go** and **Vue 3** (Vite + Tailwind CSS).

### Prerequisites
- [Go](https://go.dev) 1.24+
- [pnpm](https://pnpm.io)

### Common Commands
```bash
make dev           # Starts Go backend and Vite UI with hot-module reload
make test          # Runs all Go unit and integration tests
make seed-demo     # Populates local DB with sample workspaces & records
make build         # Builds production binary with embedded UI assets
make lint          # Runs gofmt, go vet, staticcheck, and deadcode checks
```

</details>

---

## Tech Stack

| Layer | Technology | Rationale |
|---|---|---|
| **Backend** | Go | High-performance, single compiled binary, minimal memory footprint (~25 MB). |
| **Database** | SQLite (Pure Go driver) | Zero server configuration, file-based, dependable ACID storage. |
| **Frontend** | Vue 3 + Tailwind CSS | Fast reactive UI, smooth drag-and-drop, compact bundle size. |
| **Scripting** | GopherLua | Embedded sandbox for custom triggers without external language runtimes. |
| **Agent Interface** | Model Context Protocol (MCP) | Direct, standardized AI agent integration over Server-Sent Events. |

---

## Contributing & Community

Contributions, feature ideas, and bug reports are welcome!
- Check out [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines and testing conventions.
- Please review our [Code of Conduct](CODE_OF_CONDUCT.md) and [Security Policy](SECURITY.md).

---

## License

Degubase is open-source software licensed under the [GNU General Public License v3.0](LICENSE).
