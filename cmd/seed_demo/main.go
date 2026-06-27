// cmd/seed_demo populates a DeguBase database with six curated demo workspaces
// for marketing screenshots. Designed to run against the e2e database.
//
// Usage:
//
//	go run ./cmd/seed_demo [--data-dir e2e/e2e-data]
package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/saintedlama/degubase/internal/identity"
	"github.com/saintedlama/degubase/internal/infrastructure/store"
	"github.com/saintedlama/degubase/internal/models"
	"github.com/saintedlama/degubase/internal/schema"
	"github.com/spf13/pflag"

	_ "modernc.org/sqlite"
)

//go:embed images
var imageFS embed.FS

// fileRef holds the data that goes into an image/file cell in a row.
type fileRef struct {
	fileID   string
	filename string
	size     int
}

func (f fileRef) cell() map[string]any {
	return map[string]any{
		"fileId":   f.fileID,
		"filename": f.filename,
		"size":     f.size,
		"mimeType": "image/jpeg",
	}
}

// seedRecipeImages copies the embedded food photos into {dataDir}/files/ and
// returns one fileRef per image in the same order as the recipe rows.
func seedRecipeImages(dataDir string) ([]fileRef, error) {
	filesDir := filepath.Join(dataDir, "files")
	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		return nil, err
	}
	names := []string{
		"carbonara", "pad-thai", "avocado-toast", "chicken-tacos",
		"greek-salad", "butter-chicken", "croque-madame", "pancakes",
		"ramen", "pizza",
	}
	refs := make([]fileRef, 0, len(names))
	for _, name := range names {
		data, err := imageFS.ReadFile("images/" + name + ".jpg")
		if err != nil {
			return nil, fmt.Errorf("read embedded image %s: %w", name, err)
		}
		id := uuid.New().String()
		if err := os.WriteFile(filepath.Join(filesDir, id+".jpg"), data, 0o644); err != nil {
			return nil, err
		}
		refs = append(refs, fileRef{fileID: id, filename: name + ".jpg", size: len(data)})
	}
	return refs, nil
}

type colSpec struct {
	name    string
	typ     models.ColumnType
	choices []string
}

type viewSpec struct {
	name     string
	viewType models.ViewType
	config   *models.ViewConfig
}

type tableSpec struct {
	name       string
	cols       []colSpec
	rows       [][]any
	extraViews []viewSpec
}

type workspaceSpec struct {
	name   string
	tables []tableSpec
}

func main() {
	dataDir := pflag.String("data-dir", "e2e/e2e-data", "data directory (same as server --data-dir)")
	pflag.Parse()

	if err := os.MkdirAll(filepath.Join(*dataDir, "db"), 0o755); err != nil {
		log.Fatalf("create db dir: %v", err)
	}

	dbPath := filepath.Join(*dataDir, "db", "degubase.db")
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", dbPath)

	s, err := store.New(dsn)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	rawDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("open raw db: %v", err)
	}
	rawDB.SetMaxOpenConns(1)
	defer rawDB.Close()

	recipeImgs, err := seedRecipeImages(*dataDir)
	if err != nil {
		log.Fatalf("seed recipe images: %v", err)
	}

	ctx := context.Background()
	ident := identity.NewSQLite(s.DB())
	schem := schema.NewSQLite(s.DB())

	for _, wspec := range demoWorkspaces(recipeImgs) {
		ws, err := ident.CreateWorkspace(ctx, wspec.name, "", "")
		if err != nil {
			log.Fatalf("create workspace %q: %v", wspec.name, err)
		}
		fmt.Printf("workspace %q  code=%s\n", ws.Name, ws.Code)

		for _, tspec := range wspec.tables {
			tbl, err := schem.CreateTable(ctx, ws.ID, tspec.name, "", "", "")
			if err != nil {
				log.Fatalf("create table %q: %v", tspec.name, err)
			}

			cols := make([]*models.Column, 0, len(tspec.cols))
			for _, cs := range tspec.cols {
				col, err := schem.CreateColumn(ctx, tbl.ID, cs.name, cs.typ, selectOptions(cs.choices))
				if err != nil {
					log.Fatalf("create column %q on %q: %v", cs.name, tspec.name, err)
				}
				cols = append(cols, col)
			}

			if err := insertRows(ctx, rawDB, tbl.ID, cols, tspec.rows); err != nil {
				log.Fatalf("insert rows for %q: %v", tspec.name, err)
			}

			for _, vs := range tspec.extraViews {
				if _, err := schem.CreateView(ctx, tbl.ID, vs.name, vs.viewType, vs.config); err != nil {
					log.Fatalf("create view %q on %q: %v", vs.name, tspec.name, err)
				}
			}

			fmt.Printf("  %-18s  %d rows  %d extra views\n", tspec.name, len(tspec.rows), len(tspec.extraViews))
		}
	}
}

func insertRows(ctx context.Context, db *sql.DB, tableID int64, cols []*models.Column, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO rows (table_id, data, created_at, updated_at) VALUES (?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, row := range rows {
		data := make(map[string]any, len(cols))
		for i, col := range cols {
			data[strconv.FormatInt(col.ID, 10)] = row[i]
		}
		b, _ := json.Marshal(data)
		if _, err := stmt.ExecContext(ctx, tableID, string(b), now, now); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func selectOptions(choices []string) json.RawMessage {
	if len(choices) == 0 {
		return nil
	}
	b, _ := json.Marshal(map[string]any{"choices": choices})
	return b
}

// ── Workspace definitions ──────────────────────────────────────────────────────

func demoWorkspaces(recipeImgs []fileRef) []workspaceSpec {
	return []workspaceSpec{
		bookmarksWorkspace(),
		taskTrackerWorkspace(),
		bugTrackerWorkspace(),
		crmWorkspace(),
		recipeBookWorkspace(recipeImgs),
		contentCalendarWorkspace(),
	}
}

// ── 1. Bookmarks ──────────────────────────────────────────────────────────────

func bookmarksWorkspace() workspaceSpec {
	return workspaceSpec{
		name: "Bookmarks",
		tables: []tableSpec{
			{
				name: "Links",
				cols: []colSpec{
					{name: "Title", typ: models.ColumnTypeText},
					{name: "URL", typ: models.ColumnTypeURL},
					{name: "Description", typ: models.ColumnTypeLongText},
					{name: "Tags", typ: models.ColumnTypeMultiSelect, choices: []string{"Tech", "Design", "Science", "Business", "Fun", "Dev Tools", "AI"}},
					{name: "Status", typ: models.ColumnTypeSingleSelect, choices: []string{"To Read", "Reading", "Done"}},
					{name: "Rating", typ: models.ColumnTypeRating},
				},
				rows: [][]any{
					{"Effective Go", "https://go.dev/doc/effective_go", "The official guide to writing idiomatic, production-quality Go code.", []string{"Tech", "Dev Tools"}, "Done", 5},
					{"CSS Grid Complete Guide", "https://css-tricks.com/snippets/css/complete-guide-grid/", "The definitive reference for CSS Grid layout by CSS-Tricks.", []string{"Design", "Tech"}, "Done", 4},
					{"The Feynman Technique", "https://fs.blog/feynman-technique/", "How to learn anything deeply by explaining it simply.", []string{"Science", "Fun"}, "Done", 5},
					{"Vue 3 Composition API FAQ", "https://vuejs.org/guide/extras/composition-api-faq", "Deep dive into the Composition API and when to use it over Options API.", []string{"Tech", "Dev Tools"}, "Reading", 4},
					{"SQLite is Not a Toy Database", "https://antonz.org/sqlite-is-not-a-toy-database/", "SQLite's surprising capabilities for serious production workloads.", []string{"Tech", "Dev Tools"}, "Done", 5},
					{"The Pragmatic Programmer", "https://pragprog.com/titles/tpp20/", "Essential reading for software craftspeople. Hunt & Thomas classic.", []string{"Tech", "Business"}, "To Read", 0},
					{"Building a Second Brain", "https://www.buildingasecondbrain.com/", "Tiago Forte's system for capturing and organizing digital knowledge.", []string{"Business", "Fun"}, "To Read", 0},
					{"Paul Graham Essays", "http://paulgraham.com/articles.html", "Timeless insights on startups, technology, and thinking clearly.", []string{"Business", "Fun"}, "Reading", 4},
					{"Tailwind CSS Docs", "https://tailwindcss.com/docs", "Utility-first CSS framework. Once you try it you can't go back.", []string{"Design", "Tech"}, "Done", 3},
					{"Architecture of Open Source Applications", "https://aosabook.org/en/", "How major open source projects are designed and why.", []string{"Tech", "Science"}, "To Read", 0},
					{"Designing Data-Intensive Applications", "https://dataintensive.net/", "Martin Kleppmann's tour de force on distributed systems and databases.", []string{"Tech", "Business"}, "Reading", 5},
					{"Hacker News", "https://news.ycombinator.com/", "The best link aggregator for intellectually curious builders.", []string{"Tech", "Fun"}, "Done", 4},
				},
				extraViews: []viewSpec{
					{name: "By Status", viewType: models.ViewTypeKanban, config: &models.ViewConfig{XCol: "status"}},
					{name: "Gallery", viewType: models.ViewTypeCard, config: nil},
				},
			},
		},
	}
}

// ── 2. Task Tracker ───────────────────────────────────────────────────────────

func taskTrackerWorkspace() workspaceSpec {
	return workspaceSpec{
		name: "Task Tracker",
		tables: []tableSpec{
			{
				name: "Projects",
				cols: []colSpec{
					{name: "Name", typ: models.ColumnTypeText},
					{name: "Status", typ: models.ColumnTypeSingleSelect, choices: []string{"Planning", "Active", "On Hold", "Completed"}},
					{name: "Description", typ: models.ColumnTypeLongText},
					{name: "Due Date", typ: models.ColumnTypeDate},
					{name: "Owner", typ: models.ColumnTypeText},
					{name: "Progress", typ: models.ColumnTypePercent},
				},
				rows: [][]any{
					{"Website Relaunch", "Active", "Redesign and rebuild the company website with a modern stack and improved UX.", "2026-08-01", "Alice Chen", 45},
					{"Mobile App v2", "Planning", "Next major version with offline support, push notifications, and redesigned navigation.", "2026-10-15", "Bob Martinez", 10},
					{"API Integration Hub", "Active", "Build integrations with third-party services including webhooks, OAuth, and event streaming.", "2026-07-30", "Dave Park", 60},
					{"Data Pipeline", "Completed", "Automated ETL pipeline for customer analytics and reporting dashboards.", "2026-04-30", "Carlos Reyes", 100},
					{"Design System", "Active", "Unified component library, design tokens, and Figma integration.", "2026-09-01", "Emma Wilson", 35},
				},
			},
			{
				name: "Tasks",
				cols: []colSpec{
					{name: "Title", typ: models.ColumnTypeText},
					{name: "Project", typ: models.ColumnTypeText},
					{name: "Status", typ: models.ColumnTypeSingleSelect, choices: []string{"Backlog", "Todo", "In Progress", "Done", "Blocked"}},
					{name: "Priority", typ: models.ColumnTypeSingleSelect, choices: []string{"Critical", "High", "Medium", "Low"}},
					{name: "Due Date", typ: models.ColumnTypeDate},
					{name: "Assignee", typ: models.ColumnTypeText},
					{name: "Notes", typ: models.ColumnTypeMarkdown},
				},
				rows: [][]any{
					{"Design homepage wireframes", "Website Relaunch", "In Progress", "High", "2026-06-20", "Emma Wilson", "## Brief\nFocus on hero section and feature highlights. Mobile-first."},
					{"Set up Next.js project", "Website Relaunch", "Done", "High", "2026-06-10", "Alice Chen", "Bootstrapped with App Router and Tailwind."},
					{"Write content strategy", "Website Relaunch", "Todo", "Medium", "2026-06-30", "Sarah Kim", ""},
					{"SEO audit of current site", "Website Relaunch", "Backlog", "Low", "2026-07-15", "Sarah Kim", ""},
					{"Define API authentication flow", "API Integration Hub", "Done", "Critical", "2026-06-01", "Dave Park", "Using OAuth2 + PKCE for browser flows."},
					{"Build webhook dispatcher", "API Integration Hub", "In Progress", "High", "2026-07-10", "Carlos Reyes", "## Progress\n- [x] Queue design\n- [x] Retry logic\n- [ ] Dead letter queue"},
					{"Write integration tests", "API Integration Hub", "Todo", "High", "2026-07-20", "Dave Park", ""},
					{"Document API endpoints", "API Integration Hub", "Blocked", "Medium", "2026-07-25", "Alice Chen", "Blocked on webhook dispatcher completion."},
					{"User research sessions", "Mobile App v2", "Todo", "High", "2026-08-05", "Sarah Kim", "5 sessions planned with existing power users."},
					{"Prototype offline sync", "Mobile App v2", "Backlog", "Critical", "2026-09-01", "Bob Martinez", ""},
					{"Define component tokens", "Design System", "In Progress", "High", "2026-07-01", "Emma Wilson", "## Tokens\n- Color scales\n- Spacing scale\n- Typography"},
					{"Build button variants", "Design System", "Todo", "Medium", "2026-07-15", "Emma Wilson", ""},
					{"Accessibility audit", "Design System", "Backlog", "High", "2026-08-01", "Carlos Reyes", "WCAG 2.1 AA target."},
					{"Set up ETL pipeline", "Data Pipeline", "Done", "Critical", "2026-03-15", "Carlos Reyes", ""},
					{"Write data validation layer", "Data Pipeline", "Done", "High", "2026-04-01", "Dave Park", ""},
					{"Dashboard integration", "Data Pipeline", "Done", "Medium", "2026-04-20", "Alice Chen", ""},
					{"Performance benchmarks", "API Integration Hub", "Todo", "Medium", "2026-07-30", "Dave Park", ""},
					{"Mobile navigation redesign", "Mobile App v2", "Backlog", "High", "2026-08-20", "Emma Wilson", ""},
				},
				extraViews: []viewSpec{
					{name: "Board", viewType: models.ViewTypeKanban, config: &models.ViewConfig{XCol: "status"}},
				},
			},
		},
	}
}

// ── 3. Bug Tracker ────────────────────────────────────────────────────────────

func bugTrackerWorkspace() workspaceSpec {
	return workspaceSpec{
		name: "Bug Tracker",
		tables: []tableSpec{
			{
				name: "Bugs",
				cols: []colSpec{
					{name: "Title", typ: models.ColumnTypeText},
					{name: "Severity", typ: models.ColumnTypeSingleSelect, choices: []string{"Critical", "High", "Medium", "Low"}},
					{name: "Status", typ: models.ColumnTypeSingleSelect, choices: []string{"Open", "In Progress", "Fixed", "Won't Fix", "Duplicate"}},
					{name: "Reporter", typ: models.ColumnTypeText},
					{name: "Assignee", typ: models.ColumnTypeText},
					{name: "Version", typ: models.ColumnTypeText},
					{name: "Steps", typ: models.ColumnTypeMarkdown},
				},
				rows: [][]any{
					{"App crashes on image upload > 5MB", "Critical", "Fixed", "Sarah Kim", "Dave Park", "v1.9.5", "## Steps\n1. Open any record with an image column\n2. Upload a file larger than 5MB\n3. App throws 500 and crashes\n\n**Expected:** Friendly file-size error message"},
					{"Kanban drag-drop broken on Firefox", "High", "In Progress", "Bob Martinez", "Carlos Reyes", "v2.0.0", "## Steps\n1. Open a Kanban view in Firefox 124+\n2. Drag any card to another column\n3. Card snaps back to original column\n\n**Expected:** Card moves and persists"},
					{"Date column shows wrong timezone", "High", "Open", "Alice Chen", "Dave Park", "v2.0.0", "## Steps\n1. Create a date column\n2. Enter a date while in UTC+5 timezone\n3. Observe date shows one day earlier\n\n**Expected:** Date stored as entered"},
					{"CSV export missing row-link values", "Medium", "Open", "Emma Wilson", "", "v2.0.0", "## Steps\n1. Add a row-link column to a table\n2. Link several rows\n3. Export as CSV\n4. Row-link column is empty in export"},
					{"Table slow to load with 10k+ rows", "High", "In Progress", "Dave Park", "Carlos Reyes", "v2.0.0", "## Steps\n1. Navigate to a table with 10,000+ rows\n2. Observe 8+ second load time\n\n**Expected:** < 1 second with virtual scrolling"},
					{"Markdown broken in card view", "Medium", "Fixed", "Sarah Kim", "Emma Wilson", "v1.9.5", "## Steps\n1. Add a markdown column with content\n2. Switch to Card view\n3. Raw markdown is rendered as text"},
					{"Row delete missing keyboard confirmation", "Low", "Won't Fix", "Bob Martinez", "", "v2.0.0", "## Steps\n1. Select a row\n2. Press Delete key\n3. Row deleted immediately with no confirmation\n\n**Note:** By design for power users, but needs a setting"},
					{"Search ignores long-text columns", "Medium", "Open", "Alice Chen", "Dave Park", "v2.0.0", "## Steps\n1. Add a long-text column with unique content\n2. Use global search for that content\n3. No results returned"},
					{"Auth token valid after logout", "Critical", "Fixed", "Carlos Reyes", "Dave Park", "v1.9.0", "## Steps\n1. Login and copy the JWT from DevTools\n2. Click logout\n3. Make API request with old token\n4. Request succeeds — token not invalidated\n\n**Security issue.**"},
					{"Multi-select filter uses AND instead of OR", "Medium", "In Progress", "Emma Wilson", "Alice Chen", "v2.0.0", "## Steps\n1. Filter a table by a multi-select column\n2. Select two values (e.g. 'Red' and 'Blue')\n3. Only rows with BOTH values shown\n\n**Expected:** Rows with Red OR Blue"},
					{"Image thumbnail not refreshed on replace", "Low", "Open", "Sarah Kim", "", "v2.0.0", "## Steps\n1. Upload an image to a record\n2. Delete and upload a different image\n3. Old thumbnail still shows in grid view"},
					{"Row-link circular reference causes API loop", "Critical", "Fixed", "Dave Park", "Carlos Reyes", "v1.9.5", "## Steps\n1. Create table T1 with a row-link to itself\n2. Link row A → row A\n3. Call GET /rows/{id}/graph\n4. API hangs indefinitely"},
					{"Column reorder lost on page refresh", "Medium", "Open", "Bob Martinez", "", "v2.0.0", "## Steps\n1. Drag columns to reorder them\n2. Refresh the page\n3. Columns reset to original order"},
					{"Script timeout failure is silent", "High", "Open", "Alice Chen", "Dave Park", "v2.0.0", "## Steps\n1. Write a Lua script with an infinite loop\n2. Trigger the script\n3. Script times out but no error shown to user in UI"},
					{"Emoji column corrupts CSV import", "Low", "Duplicate", "Emma Wilson", "", "v2.0.0", "## Steps\n1. Import a CSV with an emoji column\n2. Observe garbled characters in the column\n\nDuplicate of #8 (encoding issue)"},
				},
				extraViews: []viewSpec{
					{name: "Board", viewType: models.ViewTypeKanban, config: &models.ViewConfig{XCol: "status"}},
					{name: "By Severity", viewType: models.ViewTypeKanban, config: &models.ViewConfig{XCol: "severity"}},
				},
			},
			{
				name: "Releases",
				cols: []colSpec{
					{name: "Version", typ: models.ColumnTypeText},
					{name: "Status", typ: models.ColumnTypeSingleSelect, choices: []string{"Planned", "RC", "Released"}},
					{name: "Release Date", typ: models.ColumnTypeDate},
					{name: "Notes", typ: models.ColumnTypeMarkdown},
					{name: "Bug Fixes", typ: models.ColumnTypeNumber},
				},
				rows: [][]any{
					{"v2.1.0", "Planned", "2026-07-15", "## Planned\n- Fix Kanban drag-drop on Firefox\n- Fix date timezone handling\n- Fix CSV row-link export\n- Fix multi-select filter logic", 8},
					{"v2.0.1", "RC", "2026-06-20", "## RC Changes\n- Virtual scrolling for large tables\n- Multi-select filter AND→OR fix\n- Column reorder persistence", 3},
					{"v2.0.0", "Released", "2026-06-01", "## Highlights\n- **Timeline view** — date-based gantt-style view\n- **Row linking** — connect records across tables\n- **Scripting engine** — Lua scripts on row events\n- SSE real-time updates", 12},
					{"v1.9.5", "Released", "2026-05-15", "## Highlights\n- Image upload size fix\n- Markdown rendering in card view\n- Circular reference protection in row-link graph", 6},
					{"v1.9.0", "Released", "2026-04-20", "## Highlights\n- Card / gallery view\n- Auth token invalidation on logout\n- Markdown column type", 9},
					{"v1.8.3", "Released", "2026-03-10", "## Patch\n- Security hardening\n- Performance improvements to tabular view", 4},
				},
			},
		},
	}
}

// ── 4. CRM ────────────────────────────────────────────────────────────────────

func crmWorkspace() workspaceSpec {
	return workspaceSpec{
		name: "CRM",
		tables: []tableSpec{
			{
				name: "Contacts",
				cols: []colSpec{
					{name: "Name", typ: models.ColumnTypeText},
					{name: "Company", typ: models.ColumnTypeText},
					{name: "Email", typ: models.ColumnTypeEmail},
					{name: "Website", typ: models.ColumnTypeURL},
					{name: "Role", typ: models.ColumnTypeText},
					{name: "Stage", typ: models.ColumnTypeSingleSelect, choices: []string{"Lead", "Qualified", "Proposal", "Negotiation", "Won", "Lost"}},
					{name: "Deal Value", typ: models.ColumnTypeCurrency},
					{name: "Tags", typ: models.ColumnTypeMultiSelect, choices: []string{"Enterprise", "SMB", "Startup", "Partner", "Agency"}},
				},
				rows: [][]any{
					{"Jennifer Adams", "Acme Corp", "j.adams@acme.com", "https://acme.com", "VP Engineering", "Won", 84000.0, []string{"Enterprise"}},
					{"Michael Torres", "TechFlow", "m.torres@techflow.io", "https://techflow.io", "CTO", "Qualified", 12000.0, []string{"Startup"}},
					{"Lisa Zhang", "DataSphere", "l.zhang@datasphere.com", "https://datasphere.com", "Head of Product", "Proposal", 45000.0, []string{"Enterprise"}},
					{"James Wilson", "CloudBase", "j.wilson@cloudbase.net", "https://cloudbase.net", "CEO", "Won", 28000.0, []string{"SMB"}},
					{"Maria Santos", "InnovateLab", "m.santos@innovatelab.co", "https://innovatelab.co", "Engineering Lead", "Negotiation", 36000.0, []string{"Startup"}},
					{"Robert Kim", "PeakSystems", "r.kim@peaksys.com", "https://peaksys.com", "CTO", "Won", 92000.0, []string{"Partner"}},
					{"Emily Chen", "SwiftDev", "e.chen@swiftdev.io", "https://swiftdev.io", "Product Manager", "Lead", 8000.0, []string{"Startup"}},
					{"Daniel Brown", "MegaCorp", "d.brown@megacorp.com", "https://megacorp.com", "Director of IT", "Proposal", 150000.0, []string{"Enterprise"}},
					{"Sophie Williams", "NexGen", "s.williams@nexgen.co", "https://nexgen.co", "CEO", "Qualified", 22000.0, []string{"SMB"}},
					{"Alex Rivera", "BuildFast", "a.rivera@buildfast.io", "https://buildfast.io", "Lead Developer", "Lead", 6000.0, []string{"Startup"}},
					{"Priya Patel", "DigitalMind", "p.patel@digitalmind.com", "https://digitalmind.com", "COO", "Negotiation", 67000.0, []string{"Enterprise", "Agency"}},
					{"Tom Nakamura", "Stackify", "t.nakamura@stackify.io", "https://stackify.io", "Founder", "Lost", 15000.0, []string{"Startup"}},
				},
				extraViews: []viewSpec{
					{name: "Pipeline", viewType: models.ViewTypeKanban, config: &models.ViewConfig{XCol: "stage"}},
					{name: "Cards", viewType: models.ViewTypeCard, config: nil},
				},
			},
			{
				name: "Interactions",
				cols: []colSpec{
					{name: "Contact", typ: models.ColumnTypeText},
					{name: "Date", typ: models.ColumnTypeDate},
					{name: "Type", typ: models.ColumnTypeSingleSelect, choices: []string{"Email", "Call", "Meeting", "Demo", "Follow-up"}},
					{name: "Notes", typ: models.ColumnTypeMarkdown},
					{name: "Next Step", typ: models.ColumnTypeText},
				},
				rows: [][]any{
					{"Jennifer Adams", "2026-05-10", "Demo", "Walked through the full product. Strong interest in team collaboration features.", "Send enterprise pricing by Friday"},
					{"Jennifer Adams", "2026-05-17", "Email", "Sent pricing proposal. Jennifer confirmed budget approval for Q3.", "Schedule contract call"},
					{"Michael Torres", "2026-05-22", "Call", "30-min intro call. CTO-level interest, wants to see API integration story.", "Send API docs and sample integration"},
					{"Lisa Zhang", "2026-05-28", "Meeting", "On-site demo with the product team. Very positive reaction to linked records.", "Follow up with security questionnaire"},
					{"Lisa Zhang", "2026-06-03", "Email", "Sent security questionnaire. Waiting on IT review.", "Follow up in 2 weeks"},
					{"James Wilson", "2026-04-15", "Demo", "Short online demo. Liked the simplicity. Closed same week.", "Onboarding call"},
					{"Maria Santos", "2026-06-01", "Meeting", "Deep-dive on scripting engine. Engineering team very excited about Lua automation.", "Send pricing for 10 seats"},
					{"Maria Santos", "2026-06-08", "Follow-up", "Sent pricing. Maria is in final budget approval stage.", "Check in next Monday"},
					{"Daniel Brown", "2026-05-30", "Demo", "Enterprise demo with 5 stakeholders. IT director asked about SSO.", "Prepare SSO roadmap and send"},
					{"Daniel Brown", "2026-06-05", "Call", "SSO roadmap call. Timeline agreed. Moving to formal proposal.", "Send SOW"},
					{"Sophie Williams", "2026-06-02", "Email", "Inbound from website. NexGen is evaluating us vs Notion.", "Competitive overview call"},
					{"Sophie Williams", "2026-06-09", "Call", "Competitive call. Highlighted CSV import, API, and single-binary deploy.", "Trial account setup"},
					{"Alex Rivera", "2026-06-07", "Email", "Inbound dev inquiry. Solo developer use case.", "Send self-hosted docs"},
					{"Priya Patel", "2026-06-04", "Meeting", "Discovery call. Evaluating for a client project. Agency use case.", "Custom pricing proposal"},
					{"Robert Kim", "2026-03-20", "Demo", "Partner deal. PeakSystems reselling to their clients.", "Sign partner agreement"},
				},
			},
		},
	}
}

// ── 5. Recipe Book ────────────────────────────────────────────────────────────

func recipeBookWorkspace(imgs []fileRef) workspaceSpec {
	// imgs order matches recipe row order: carbonara, pad-thai, avocado-toast,
	// chicken-tacos, greek-salad, butter-chicken, croque-madame, pancakes, ramen, pizza
	img := func(i int) map[string]any { return imgs[i].cell() }

	return workspaceSpec{
		name: "Recipe Book",
		tables: []tableSpec{
			{
				name: "Recipes",
				cols: []colSpec{
					// Image first so it renders as a cover photo in card view
					{name: "Photo", typ: models.ColumnTypeImage},
					{name: "Name", typ: models.ColumnTypeText},
					{name: "Cuisine", typ: models.ColumnTypeSingleSelect, choices: []string{"Italian", "Asian", "American", "Mexican", "Mediterranean", "Indian", "French"}},
					{name: "Difficulty", typ: models.ColumnTypeSingleSelect, choices: []string{"Easy", "Medium", "Hard"}},
					{name: "Prep Time", typ: models.ColumnTypeNumber},
					{name: "Rating", typ: models.ColumnTypeRating},
					{name: "Tags", typ: models.ColumnTypeMultiSelect, choices: []string{"Vegetarian", "Vegan", "Gluten-Free", "Quick", "Comfort Food", "Party"}},
					{name: "Ingredients", typ: models.ColumnTypeLongText},
					{name: "Instructions", typ: models.ColumnTypeMarkdown},
				},
				rows: [][]any{
					{img(0), "Spaghetti Carbonara", "Italian", "Medium", 30, 5, []string{"Comfort Food"},
						"400g spaghetti, 200g pancetta, 4 egg yolks, 100g pecorino romano, black pepper",
						"## Instructions\n1. Cook spaghetti al dente in salted water\n2. Fry pancetta until crispy\n3. Whisk egg yolks with grated pecorino\n4. Toss hot pasta with pancetta off heat\n5. Add egg mixture, toss vigorously with pasta water\n6. Season with black pepper and serve immediately"},
					{img(1), "Pad Thai", "Asian", "Medium", 25, 4, []string{"Quick", "Gluten-Free"},
						"200g rice noodles, 200g shrimp, 2 eggs, bean sprouts, spring onions, peanuts, tamarind paste, fish sauce, palm sugar",
						"## Instructions\n1. Soak noodles in warm water 30 minutes\n2. Stir-fry shrimp in hot wok\n3. Push aside, scramble eggs\n4. Add noodles and sauce (tamarind, fish sauce, sugar)\n5. Toss with bean sprouts and spring onions\n6. Serve with crushed peanuts and lime"},
					{img(2), "Avocado Toast", "American", "Easy", 10, 4, []string{"Vegetarian", "Quick"},
						"2 slices sourdough, 1 ripe avocado, lemon juice, chili flakes, sea salt, olive oil",
						"## Instructions\n1. Toast bread until golden\n2. Mash avocado with lemon juice and salt\n3. Spread on toast\n4. Drizzle olive oil, sprinkle chili flakes"},
					{img(3), "Chicken Tacos", "Mexican", "Easy", 20, 5, []string{"Quick"},
						"6 corn tortillas, 400g chicken breast, cumin, paprika, lime, salsa, sour cream, cilantro",
						"## Instructions\n1. Season chicken with cumin, paprika, salt\n2. Cook in skillet 6 min each side\n3. Rest and slice\n4. Warm tortillas in dry pan\n5. Assemble with salsa, sour cream, cilantro, lime"},
					{img(4), "Greek Salad", "Mediterranean", "Easy", 15, 4, []string{"Vegetarian", "Vegan", "Gluten-Free", "Quick"},
						"1 cucumber, 3 tomatoes, 1 red onion, kalamata olives, 200g feta, olive oil, oregano",
						"## Instructions\n1. Chop cucumber and tomatoes into chunks\n2. Slice red onion thinly\n3. Combine with olives and feta\n4. Dress with olive oil, salt, oregano\n5. Let sit 10 min before serving"},
					{img(5), "Butter Chicken", "Indian", "Hard", 60, 5, []string{"Comfort Food", "Party"},
						"800g chicken thighs, 400ml tomato passata, 200ml cream, 2 onions, ginger, garlic, garam masala, fenugreek",
						"## Instructions\n1. Marinate chicken in yogurt and spices 2 hours\n2. Char chicken in high heat pan\n3. Sauté onion, ginger, garlic until golden\n4. Add passata and spices, simmer 20 min\n5. Blend sauce smooth, return to pan\n6. Add cream and chicken, simmer 15 min"},
					{img(6), "Croque Madame", "French", "Medium", 20, 4, []string{"Comfort Food"},
						"4 slices white bread, 4 slices ham, 100g Gruyère, 200ml béchamel, 2 eggs, butter, Dijon mustard",
						"## Instructions\n1. Make béchamel: butter, flour, milk, nutmeg\n2. Spread mustard on bread, add ham and Gruyère\n3. Top with béchamel and more Gruyère\n4. Broil 5 min until golden\n5. Fry egg sunny-side up and place on top"},
					{img(7), "Banana Pancakes", "American", "Easy", 15, 5, []string{"Vegetarian", "Quick", "Gluten-Free"},
						"2 ripe bananas, 2 eggs, 50g rolled oats, pinch of cinnamon, coconut oil",
						"## Instructions\n1. Mash bananas in a bowl\n2. Whisk in eggs\n3. Blend in oats and cinnamon\n4. Cook small pancakes in coconut oil 2 min per side\n5. Serve with maple syrup or berries"},
					{img(8), "Miso Ramen", "Asian", "Hard", 90, 5, []string{"Comfort Food"},
						"200g ramen noodles, 1L chicken stock, 3 tbsp white miso, soy sauce, chashu pork belly, soft-boiled egg, nori, spring onions, sesame oil",
						"## Instructions\n1. Slow-cook pork belly in soy, mirin, sake 2 hours\n2. Soft-boil eggs and marinate in soy broth\n3. Heat stock, whisk in miso and soy sauce\n4. Cook noodles separately\n5. Assemble: noodles, hot broth, sliced chashu, halved egg\n6. Garnish with nori, spring onions, sesame oil"},
					{img(9), "Margherita Pizza", "Italian", "Medium", 45, 5, []string{"Vegetarian", "Party"},
						"Pizza dough, 400g San Marzano tomatoes, 250g buffalo mozzarella, fresh basil, olive oil, sea salt",
						"## Instructions\n1. Stretch dough to thin round\n2. Crush tomatoes by hand, season with salt\n3. Spread tomato base, leaving border\n4. Tear mozzarella and distribute\n5. Bake at max oven temp (250°C) for 10-12 min\n6. Top with fresh basil and drizzle of olive oil"},
				},
				extraViews: []viewSpec{
					{name: "Gallery", viewType: models.ViewTypeCard, config: &models.ViewConfig{
						CardFields: []models.CardFieldConfig{
							{Col: "photo", ShowLabel: false},
							{Col: "name", ShowLabel: false},
							{Col: "cuisine", ShowLabel: true},
							{Col: "difficulty", ShowLabel: true},
							{Col: "prep-time", ShowLabel: true},
							{Col: "rating", ShowLabel: false},
							{Col: "tags", ShowLabel: true},
						},
					}},
				},
			},
		},
	}
}

// ── 6. Content Calendar ───────────────────────────────────────────────────────

func contentCalendarWorkspace() workspaceSpec {
	return workspaceSpec{
		name: "Content Calendar",
		tables: []tableSpec{
			{
				name: "Posts",
				cols: []colSpec{
					{name: "Title", typ: models.ColumnTypeText},
					{name: "Status", typ: models.ColumnTypeSingleSelect, choices: []string{"Idea", "Draft", "Review", "Scheduled", "Published"}},
					{name: "Author", typ: models.ColumnTypeText},
					{name: "Publish Date", typ: models.ColumnTypeDate},
					{name: "Channel", typ: models.ColumnTypeSingleSelect, choices: []string{"Blog", "Twitter", "LinkedIn", "Newsletter", "YouTube"}},
					{name: "Tags", typ: models.ColumnTypeMultiSelect, choices: []string{"Tutorial", "Opinion", "News", "Case Study", "How-To"}},
					{name: "Word Count", typ: models.ColumnTypeNumber},
					{name: "Brief", typ: models.ColumnTypeMarkdown},
				},
				// 25 posts across 5 channels in a tight 5-week window (Jun 14 – Jul 18)
				// so the timeline swimlane view renders visibly full on first load.
				rows: [][]any{
					// Blog – weekly cadence
					{"Getting Started with Degubase", "Published", "Alice Chen", "2026-06-15", "Blog", []string{"Tutorial", "How-To"}, 1800, "## Brief\nIntroductory walkthrough for new users. Cover workspace creation, table setup, and first rows."},
					{"SQLite is Not a Toy Database", "Published", "Dave Park", "2026-06-22", "Blog", []string{"Opinion"}, 2100, "## Brief\nDefend the SQLite architecture decision. Address scale concerns and highlight simplicity wins."},
					{"Row Linking Deep Dive", "Scheduled", "Dave Park", "2026-06-29", "Blog", []string{"Tutorial"}, 1500, "## Brief\nExplain the row-link column type, cross-table references, same-table hierarchies, and the graph API."},
					{"Lua Scripting Cookbook", "Draft", "Carlos Reyes", "2026-07-06", "Blog", []string{"Tutorial", "How-To"}, 2200, "## Brief\nPractical Lua recipes: auto-set timestamps, webhook on create, cross-table cascades, data validation."},
					{"Self-hosting Guide: VPS + Docker", "Review", "Bob Martinez", "2026-07-13", "Blog", []string{"How-To", "Tutorial"}, 1600, "## Brief\nStep-by-step: DigitalOcean droplet, Docker Compose, reverse proxy with Caddy, auto HTTPS."},
					{"Using Degubase as Agent Memory", "Idea", "Alice Chen", "2026-07-20", "Blog", []string{"Tutorial", "Case Study"}, 0, "## Brief\nAI agent integration story: workspace tokens, context fields, OpenAPI autodiscovery, row-link traversal."},

					// Twitter – 2-3× per week
					{"What makes Degubase different — thread", "Published", "Sarah Kim", "2026-06-14", "Twitter", []string{"Opinion"}, 0, "## Brief\n8-tweet thread on the anti-platform philosophy. End with a CTA to the quick-start."},
					{"SQLite for production — hot take", "Published", "Dave Park", "2026-06-17", "Twitter", []string{"Opinion"}, 0, "## Brief\nControversial thread. Lead with a benchmark. Expect engagement from the Postgres crowd."},
					{"Weekend build: recipe book demo", "Published", "Emma Wilson", "2026-06-21", "Twitter", []string{"Tutorial"}, 0, "## Brief\nLive-build thread. Screenshots of card view. Tag it #buildinpublic."},
					{"Kanban tip of the week", "Scheduled", "Emma Wilson", "2026-06-24", "Twitter", []string{"How-To"}, 0, "## Brief\nShort tip thread. Show swimlane grouping with a GIF. 3 tweets max."},
					{"v2.1 preview — Timeline view", "Scheduled", "Sarah Kim", "2026-06-28", "Twitter", []string{"News"}, 0, "## Brief\nTeaser thread with an animated GIF of the new timeline. Link to the changelog."},
					{"One binary, one file — quote tweet", "Draft", "Alice Chen", "2026-07-02", "Twitter", []string{"Opinion"}, 0, "## Brief\nReact to a popular 'your stack is too complex' post. Keep it friendly, link to degubase."},
					{"Community spotlight", "Idea", "Sarah Kim", "2026-07-08", "Twitter", []string{"Case Study"}, 0, "## Brief\nFeature a user project built with Degubase. Need to source a good example first."},
					{"Hot take: your database is too big", "Idea", "Dave Park", "2026-07-11", "Twitter", []string{"Opinion"}, 0, "## Brief\nArgue that most side projects fit in <1GB SQLite forever. Cite some stats."},

					// LinkedIn – weekly professional
					{"Why I stopped using Notion", "Published", "Alice Chen", "2026-06-16", "LinkedIn", []string{"Opinion"}, 0, "## Brief\nPersonal story. Pain points: slow, bloated, not programmable. Position Degubase as the hacker's alternative."},
					{"5 database mistakes small teams make", "Scheduled", "Dave Park", "2026-06-23", "LinkedIn", []string{"Opinion", "How-To"}, 0, "## Brief\nThought-leadership post. Mistake #5: no schema. End with 'here's what we do instead'."},
					{"Degubase vs Airtable: honest review", "Review", "Sarah Kim", "2026-06-30", "LinkedIn", []string{"Opinion", "Case Study"}, 0, "## Brief\nFair comparison. Airtable wins on polish; Degubase wins on ownership and scripting."},
					{"How we use Kanban internally", "Draft", "Emma Wilson", "2026-07-07", "LinkedIn", []string{"How-To"}, 0, "## Brief\nBehind-the-scenes post. Show our actual task board. Makes the product feel real and used."},
					{"Open source: 6-month update", "Idea", "Alice Chen", "2026-07-14", "LinkedIn", []string{"News"}, 0, "## Brief\nMilestone post. Star count, downloads, community highlights. Authentic and grateful tone."},

					// Newsletter – bi-weekly
					{"June Edition: Timeline view ships", "Published", "Alice Chen", "2026-06-18", "Newsletter", []string{"News", "Tutorial"}, 0, "## Brief\nLead with Timeline view. Quick tutorial. Links to 3 top blog posts. Changelog summary."},
					{"July 1st: Scripting engine deep dive", "Scheduled", "Dave Park", "2026-07-01", "Newsletter", []string{"Tutorial"}, 0, "## Brief\nFeature the Lua scripting engine. Include 2 practical recipes readers can copy-paste immediately."},
					{"Mid-July: v2.1 ships", "Idea", "Sarah Kim", "2026-07-15", "Newsletter", []string{"News"}, 0, "## Brief\nRelease announcement edition. Highlight every fix in v2.1. Thank contributors by name."},

					// YouTube – bi-weekly
					{"Building a CRM in 10 Minutes", "Published", "Alice Chen", "2026-06-19", "YouTube", []string{"Tutorial", "How-To"}, 0, "## Brief\nScreen recording: workspace → tables → columns → views. End result: working contact pipeline with kanban."},
					{"Kanban Workflow Demo", "Scheduled", "Emma Wilson", "2026-07-03", "YouTube", []string{"Tutorial", "How-To"}, 0, "## Brief\nDemo the full kanban flow: drag cards, add rows from board, swimlane grouping. Keep under 8 min."},
					{"Degubase vs Notion: side-by-side", "Idea", "Sarah Kim", "2026-07-17", "YouTube", []string{"Opinion", "Case Study"}, 0, "## Brief\nScreen-split comparison. Same use case built in both. Let the viewer decide. Aim for 10 min."},
				},
				extraViews: []viewSpec{
					{name: "Timeline", viewType: models.ViewTypeTimeline, config: &models.ViewConfig{TimelineConfig: &models.TimelineConfig{DateCol: "publish-date"}, GroupByCol: "channel"}},
					{name: "Board", viewType: models.ViewTypeKanban, config: &models.ViewConfig{XCol: "status", GroupByCol: "channel"}},
				},
			},
			{
				name: "Ideas",
				cols: []colSpec{
					{name: "Title", typ: models.ColumnTypeText},
					{name: "Priority", typ: models.ColumnTypeSingleSelect, choices: []string{"Hot", "Warm", "Cool"}},
					{name: "Notes", typ: models.ColumnTypeLongText},
					{name: "Tags", typ: models.ColumnTypeMultiSelect, choices: []string{"Tutorial", "Opinion", "News", "Case Study", "How-To"}},
				},
				rows: [][]any{
					{"Benchmark: Degubase vs local Postgres", "Hot", "Could be very viral among devs. Show where SQLite wins and where it doesn't.", []string{"Opinion", "Case Study"}},
					{"Degubase on Raspberry Pi", "Warm", "DIY homelab crowd would love this. Pair with a Node-RED integration demo.", []string{"Tutorial", "How-To"}},
					{"Multi-workspace CLI tool", "Cool", "A CLI that wraps the API for power users. Could generate a lot of GitHub stars.", []string{"Tutorial"}},
					{"Interview: open source maintainer workflows", "Warm", "Talk to 3-4 open source maintainers about how they track issues. Position Degubase as alternative.", []string{"Case Study"}},
					{"Degubase + n8n automation", "Hot", "n8n has a huge community. Show webhook + Degubase script combo for automation.", []string{"Tutorial", "How-To"}},
					{"Why I switched from Notion to Degubase", "Hot", "Personal story format. Gets shared a lot. Needs a real user testimonial or first-person framing.", []string{"Opinion"}},
					{"Building a personal finance tracker", "Warm", "Use currency, date, and chart columns. Good showcase of the number column types.", []string{"Tutorial"}},
					{"The future of local-first apps", "Cool", "Philosophical piece on local-first, CRDT, and where Degubase fits. Long-form essay.", []string{"Opinion"}},
				},
			},
		},
	}
}
