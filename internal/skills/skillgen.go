package skills

import (
	"encoding/json"
	"fmt"
	"strings"

	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/models"
)

type skillGenInput struct {
	Workspace   *models.Workspace
	Tables      []models.Table
	SampleRows  map[int64][]models.Row // tableID → up to 5 rows
	BaseURL     string
	SkillName   string
	Description string
	Operations  []string
	NoAuth      bool
}

func generateSkillContent(in skillGenInput) string {
	var b strings.Builder

	code := toKebabCase(in.SkillName)

	fmt.Fprintf(&b, "---\nname: %s\ndescription: %s\n---\n\n", code, in.Description)
	fmt.Fprintf(&b, "# %s\n\n", in.SkillName)
	if in.Workspace.Context != "" {
		fmt.Fprintf(&b, "%s\n\n", in.Workspace.Context)
	}

	// Data sections — schema + sample rows per table (first so it's immediately visible)
	b.WriteString("## Data\n\n")
	for _, t := range in.Tables {
		fmt.Fprintf(&b, "### %s\n\n", t.Name)
		if t.Context != "" {
			fmt.Fprintf(&b, "%s\n\n", t.Context)
		}

		dataCols := dataColumns(t.Columns)
		if len(dataCols) > 0 {
			b.WriteString("**Columns** — use the **Code** as the JSON key in row data:\n\n")
			b.WriteString("| Code | Name | Type |\n|------|------|------|\n")
			for _, c := range dataCols {
				fmt.Fprintf(&b, "| %s | %s | %s |\n", c.Code, c.Name, c.Type)
			}
			b.WriteString("\n")
			for _, c := range dataCols {
				if c.Type == models.ColumnTypeSingleSelect || c.Type == models.ColumnTypeMultiSelect {
					if opts := parseSelectOptions(c.Options); len(opts) > 0 {
						fmt.Fprintf(&b, "Options for `%s`: %s\n\n", c.Name, strings.Join(opts, ", "))
					}
				}
				if c.Type == models.ColumnTypeChecklist {
					fmt.Fprintf(&b, "`%s` is a checklist — value is a JSON array of `{\"text\": \"...\", \"checked\": false}` objects. Example: `[{\"text\": \"First item\", \"checked\": true}, {\"text\": \"Second item\", \"checked\": false}]`\n\n", c.Name)
				}
			}
		}

		// Sample rows
		if rows := in.SampleRows[t.ID]; len(rows) > 0 && len(dataCols) > 0 {
			dispCols := dataCols
			if len(dispCols) > 6 {
				dispCols = dispCols[:6]
			}
			b.WriteString("**Sample rows:**\n\n")
			b.WriteString("| id |")
			for _, c := range dispCols {
				fmt.Fprintf(&b, " %s |", c.Name)
			}
			b.WriteString("\n|----|")
			for range dispCols {
				b.WriteString("-----|")
			}
			b.WriteString("\n")
			for _, row := range rows {
				var data map[string]interface{}
				_ = json.Unmarshal(row.Data, &data)
				fmt.Fprintf(&b, "| %d |", row.ID)
				for _, c := range dispCols {
					key := fmt.Sprintf("%d", c.ID)
					val := ""
					if v, ok := data[key]; ok && v != nil {
						raw, _ := json.Marshal(v)
						val = strings.Trim(string(raw), `"`)
						if len(val) > 30 {
							val = val[:27] + "..."
						}
					}
					fmt.Fprintf(&b, " %s |", val)
				}
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
	}

	// Endpoints per table
	b.WriteString("## Endpoints\n\n")
	for _, t := range in.Tables {
		wsCode := in.Workspace.Code
		fmt.Fprintf(&b, "### %s\n\n", t.Name)

		sampleKey := firstDataColumnCode(t.Columns)

		if hasOp(in.Operations, "read") {
			fmt.Fprintf(&b, "**List rows** — returns `{\"data\":[...],\"total\":N,\"page\":1,\"page_size\":100,\"total_pages\":N}`.\n```\nGET %s/api/workspaces/%s/tables/%s/rows\n```\n\n",
				in.BaseURL, wsCode, t.Code)

			fmt.Fprintf(&b, "**Get row** — returns `{\"id\":N,\"table_id\":N,\"data\":{...},\"created_at\":\"...\",\"updated_at\":\"...\"}`.\n```\nGET %s/api/workspaces/%s/tables/%s/rows/{rowID}\n```\n\n",
				in.BaseURL, wsCode, t.Code)

			// Filter and sort reference — always shown with read
			b.WriteString("**Filtering & sorting**\n\n")
			b.WriteString("Pagination:\n\n")
			fmt.Fprintf(&b, "- `page` — 1-based page number (default 1)\n")
			fmt.Fprintf(&b, "- `pageSize` — rows per page (default %d, max %d)\n\n", httplib.DefaultPageSize, httplib.MaxPageSize)
			b.WriteString("Filtering — repeat `filter=` for AND conditions:\n\n")
			b.WriteString("```\n")
			b.WriteString("filter=status:Open                  # implied \"is\"\n")
			b.WriteString("filter=status:is:Open               # explicit operator\n")
			b.WriteString("filter=priority:gt:5\n")
			b.WriteString("filter=tags:in:Bug|Feature          # pipe-separate values for in/not_in\n")
			b.WriteString("filter=name:contains:John&filter=tags:in:Bug|Feature\n")
			b.WriteString("```\n\n")
			b.WriteString("Sorting — comma-separate multiple columns:\n\n")
			b.WriteString("```\n")
			b.WriteString("sort=created_at                     # default asc\n")
			b.WriteString("sort=created_at:asc,priority:desc\n")
			b.WriteString("```\n\n")
			b.WriteString("Filter operators: `is`, `is_not` (exact match); `contains`, `not_contains`; `is_empty`, `is_not_empty`; `eq`, `gt`, `lt` (numbers); `before`, `after`, `on` (dates); `is_true`, `is_false` (checkbox); `in`, `not_in` (multi-select, pipe-separate values).\n\n")
			b.WriteString("Column selection — restrict which fields appear in `data` (reduces payload size):\n\n")
			b.WriteString("```\n")
			b.WriteString("select=title,status,severity        # comma-separated column codes\n")
			b.WriteString("```\n\n")

			fmt.Fprintf(&b, "**List rows grouped by a column** — returns an array of groups, each with its rows and total count. Useful for querying all records in a specific category (e.g. all open issues, all tasks in a given status).\n```\nGET %s/api/workspaces/%s/tables/%s/groups?groupBy=<col_code>\n```\n\n",
				in.BaseURL, wsCode, t.Code)
			b.WriteString("Response — array of group objects, `null` value means the field is unset:\n\n")
			b.WriteString("```json\n[\n  {\"value\": \"In progress\", \"total\": 12, \"rows\": [{...}, ...]},\n  {\"value\": \"Done\",        \"total\": 34, \"rows\": [{...}, ...]},\n  {\"value\": null,          \"total\": 3,  \"rows\": [{...}, ...]}\n]\n```\n\n")
			b.WriteString("Parameters (all optional except `groupBy`):\n\n")
			b.WriteString("- `groupBy` — **required** — column code to group by (must be a `single-select` column)\n")
			fmt.Fprintf(&b, "- `pageSize` — max rows returned per group (default %d)\n", httplib.DefaultPageSize)
			b.WriteString("- `filter` / `filters` — same syntax as List rows\n")
			b.WriteString("- `sort` — same syntax as List rows\n")
			b.WriteString("- `q` — full-text search\n\n")
		}

		if hasOp(in.Operations, "create") {
			fmt.Fprintf(&b, "**Create row** — returns 201 with the created row.\n```\nPOST %s/api/workspaces/%s/tables/%s/rows\nContent-Type: application/json\n\n{\"data\": {\"%s\": \"<value>\"}}\n```\n\n",
				in.BaseURL, wsCode, t.Code, sampleKey)
		}

		if hasOp(in.Operations, "update") {
			fmt.Fprintf(&b, "**Patch row (partial update)** — merges supplied fields; returns 200 with the updated row.\n```\nPATCH %s/api/workspaces/%s/tables/%s/rows/<row_id>\nContent-Type: application/json\n\n{\"data\": {\"%s\": \"<value>\"}}\n```\n\n",
				in.BaseURL, wsCode, t.Code, sampleKey)
		}

		if hasOp(in.Operations, "delete") {
			fmt.Fprintf(&b, "**Delete row** — returns 204 No Content.\n```\nDELETE %s/api/workspaces/%s/tables/%s/rows/<row_id>\n```\n\n",
				in.BaseURL, wsCode, t.Code)
		}

		if hasOp(in.Operations, "views") {
			fmt.Fprintf(&b, "**List views**\n```\nGET %s/api/workspaces/%s/tables/%s/views\n```\n\n",
				in.BaseURL, wsCode, t.Code)
			fmt.Fprintf(&b, "**Create view** — types: `grid` (spreadsheet), `kanban` (board), `card` (gallery).\n```\nPOST %s/api/workspaces/%s/tables/%s/views\nContent-Type: application/json\n\n{\"name\": \"<name>\", \"type\": \"grid\"}\n```\n\n",
				in.BaseURL, wsCode, t.Code)
			fmt.Fprintf(&b, "**Rename view**\n```\nPUT %s/api/workspaces/%s/tables/%s/views/<viewCode>\nContent-Type: application/json\n\n{\"name\": \"<name>\"}\n```\n\n",
				in.BaseURL, wsCode, t.Code)
			fmt.Fprintf(&b, "**Delete view** — returns 204 No Content.\n```\nDELETE %s/api/workspaces/%s/tables/%s/views/<viewCode>\n```\n\n",
				in.BaseURL, wsCode, t.Code)
		}

		if hasOp(in.Operations, "schema") {
			fmt.Fprintf(&b, "**List columns**\n```\nGET %s/api/workspaces/%s/tables/%s/columns\n```\n\n",
				in.BaseURL, wsCode, t.Code)
			fmt.Fprintf(&b, "**Add column** — `type` values: `text`, `long-text`, `number`, `single-select`, `multi-select`, `date`, `datetime`, `checkbox`, `url`, `email`, `file`, `image`, `checklist` (array of `{text,checked}` objects), `markdown`, `rating`, `percent`, `currency`.\n```\nPOST %s/api/workspaces/%s/tables/%s/columns\nContent-Type: application/json\n\n{\"name\": \"<name>\", \"type\": \"<type>\"}\n```\n\n",
				in.BaseURL, wsCode, t.Code)
			fmt.Fprintf(&b, "**Update column**\n```\nPUT %s/api/workspaces/%s/tables/%s/columns/<col_code>\nContent-Type: application/json\n\n{\"name\": \"<name>\", \"type\": \"<type>\"}\n```\n\n",
				in.BaseURL, wsCode, t.Code)
			fmt.Fprintf(&b, "**Delete column** — returns 204 No Content.\n```\nDELETE %s/api/workspaces/%s/tables/%s/columns/<col_code>\n```\n\n",
				in.BaseURL, wsCode, t.Code)
		}
	}

	// Scripting section — included when "scripting" operation is selected
	if hasOp(in.Operations, "scripting") {
		writeScriptingSection(&b, in)
	}

	// Annotations section — included when any change operation is present
	if hasOp(in.Operations, "create") || hasOp(in.Operations, "update") || hasOp(in.Operations, "delete") {
		writeAnnotationsSection(&b, in)
	}

	// Authentication
	if !in.NoAuth {
		b.WriteString("## Authentication\n\n")
		b.WriteString("Add this header to every request:\n\n")
		b.WriteString("```\nAuthorization: Bearer $DEGUBASE_TOKEN\n```\n\n")
		b.WriteString("Set `DEGUBASE_TOKEN` to your workspace API token.\n\n")
	}

	// OAS reference
	wsBaseURL := in.BaseURL + "/api/workspaces/" + in.Workspace.Code
	b.WriteString("## API Reference\n\n")
	fmt.Fprintf(&b, "Full OpenAPI spec: `%s/api/docs/swagger.json`  \n", in.BaseURL)
	fmt.Fprintf(&b, "Interactive docs: `%s/api/docs/index.html`\n\n", in.BaseURL)
	fmt.Fprintf(&b, "**Workspace code:** `%s`  \n", in.Workspace.Code)
	fmt.Fprintf(&b, "**Workspace API base URL:** `%s`  \n", wsBaseURL)
	b.WriteString("*(All table endpoints are rooted at this URL.)*\n\n")

	return b.String()
}

func dataColumns(cols []models.Column) []models.Column {
	out := make([]models.Column, 0, len(cols))
	for _, c := range cols {
		if c.Type != models.ColumnTypeCreatedAt && c.Type != models.ColumnTypeUpdatedAt {
			out = append(out, c)
		}
	}
	return out
}

func hasOp(ops []string, op string) bool {
	if len(ops) == 0 {
		return true
	}
	for _, o := range ops {
		if o == op {
			return true
		}
	}
	return false
}

func toKebabCase(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	var out strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func firstDataColumnCode(cols []models.Column) string {
	for _, c := range cols {
		if c.Type != models.ColumnTypeCreatedAt && c.Type != models.ColumnTypeUpdatedAt {
			return c.Code
		}
	}
	return "field"
}

func writeScriptingSection(b *strings.Builder, in skillGenInput) {
	wsCode := in.Workspace.Code

	b.WriteString("## Scripting\n\n")
	b.WriteString("Scripts run automatically on record events. They are written in **Lua** and have access to row data, environment variables, and helpers to read/write records and make HTTP calls.\n\n")

	b.WriteString("### Managing Scripts\n\n")
	fmt.Fprintf(b, "**List scripts**\n```\nGET %s/api/workspaces/%s/scripts\n```\n\n", in.BaseURL, wsCode)
	fmt.Fprintf(b, "**Create script**\n```\nPOST %s/api/workspaces/%s/scripts\nContent-Type: application/json\n\n{\"name\": \"My Script\", \"event_type\": \"record.created\", \"code\": \"-- Lua\", \"enabled\": true, \"table_ids\": []}\n```\n\n", in.BaseURL, wsCode)
	b.WriteString("`event_type` values:\n- `record.created` — fires after a new record is saved\n- `record.updated` — fires after a record is patched\n- `record.deleted` — fires after a record is removed\n\n")
	b.WriteString("`table_ids` — list of table IDs this script applies to; omit or pass `[]` for workspace-wide.\n\n")
	fmt.Fprintf(b, "**Update script**\n```\nPUT %s/api/workspaces/%s/scripts/{scriptID}\nContent-Type: application/json\n\n{\"name\": \"...\", \"event_type\": \"record.updated\", \"code\": \"...\", \"enabled\": true}\n```\n\n", in.BaseURL, wsCode)
	fmt.Fprintf(b, "**Delete script** — returns 204 No Content.\n```\nDELETE %s/api/workspaces/%s/scripts/{scriptID}\n```\n\n", in.BaseURL, wsCode)

	b.WriteString("### Environment Variables\n\n")
	b.WriteString("Store secrets and config outside script code:\n\n")
	fmt.Fprintf(b, "**List env vars**\n```\nGET %s/api/workspaces/%s/scripts/env\n```\n\n", in.BaseURL, wsCode)
	fmt.Fprintf(b, "**Upsert env var**\n```\nPOST %s/api/workspaces/%s/scripts/env\nContent-Type: application/json\n\n{\"key\": \"API_KEY\", \"value\": \"secret\", \"is_secret\": true}\n```\n\n", in.BaseURL, wsCode)
	fmt.Fprintf(b, "**Delete env var** — returns 204 No Content.\n```\nDELETE %s/api/workspaces/%s/scripts/env/{envID}\n```\n\n", in.BaseURL, wsCode)

	b.WriteString("### Lua Runtime\n\n")
	b.WriteString("Scripts run in a sandboxed Lua 5.1 environment. The following globals are injected before your script runs:\n\n")

	b.WriteString("**Event context:**\n\n")
	b.WriteString("| Global | Type | Description |\n|--------|------|-------------|\n")
	b.WriteString("| `event.type` | string | Triggering event (`record.created`, `record.updated`, `record.deleted`) |\n")
	b.WriteString("| `event.table_name` | string | Name of the table |\n")
	b.WriteString("| `event.table_code` | string | Code of the table |\n")
	b.WriteString("| `event.row_id` | number | ID of the triggering record |\n")
	b.WriteString("| `event.data` | table | Field values keyed by column name |\n")
	b.WriteString("| `row` | table | Alias for `event.data` |\n")
	b.WriteString("\n")

	b.WriteString("**Record access:**\n\n")
	b.WriteString("| Call | Returns | Description |\n|------|---------|-------------|\n")
	b.WriteString("| `degubase.get_row(id)` | table | Fetch record data by ID (name-keyed) |\n")
	b.WriteString("| `degubase.update_row(id, data)` | bool | Replace all data fields for a record |\n")
	b.WriteString("| `degubase.update_field(colName, value)` | bool | Patch a single field on the triggering record |\n")
	b.WriteString("| `degubase.create_row(data)` | number | Create a new record; returns new row ID |\n")
	b.WriteString("\n")

	b.WriteString("**Logging:**\n\n")
	b.WriteString("| Call | Description |\n|------|-------------|\n")
	b.WriteString("| `log.info(msg)` / `print(msg)` | Info message |\n")
	b.WriteString("| `log.warn(msg)` | Warning |\n")
	b.WriteString("| `log.error(msg)` | Error |\n")
	b.WriteString("\n")

	b.WriteString("**HTTP:**\n\n")
	b.WriteString("| Call | Returns | Description |\n|------|---------|-------------|\n")
	b.WriteString("| `http.get(url)` | `{status, body}` | GET request (10 s timeout) |\n")
	b.WriteString("| `http.post(url, jsonBody)` | `{status, body}` | POST with JSON body |\n")
	b.WriteString("\n")

	b.WriteString("**Environment variables** — injected as globals and via `env(\"KEY\")`:\n\n")
	b.WriteString("```lua\nlocal token = env(\"API_KEY\")  -- recommended\n```\n\n")

	b.WriteString("**Example:**\n\n")
	b.WriteString("```lua\n-- Set Status to \"Received\" when a record is created\nif event.type == \"record.created\" then\n  degubase.update_field(\"Status\", \"Received\")\n  log.info(\"Status set for row \" .. tostring(event.row_id))\nend\n```\n\n")
}

func writeAnnotationsSection(b *strings.Builder, in skillGenInput) {
	wsCode := in.Workspace.Code

	b.WriteString("## Annotations\n\n")
	b.WriteString("Annotations let you attach markdown notes to a record's activity feed. Use them to document what you did, explain decisions, or propose next steps during a refinement step.\n\n")
	b.WriteString("An annotation appears as a timestamped entry in the record's Activity panel alongside field-change history. The `annotation` field accepts **markdown**.\n\n")

	b.WriteString("### Endpoints\n\n")
	for _, t := range in.Tables {
		base := fmt.Sprintf("%s/api/workspaces/%s/tables/%s/rows", in.BaseURL, wsCode, t.Code)
		fmt.Fprintf(b, "**%s** — base: `%s/{rowID}`\n\n", t.Name, base)

		fmt.Fprintf(b, "**List activity** (field-change entries + annotations)\n```\nGET %s/{rowID}/history\n```\n\n", base)
		b.WriteString("Response — array of entries, newest first:\n\n")
		b.WriteString("```json\n[\n  {\"id\": 1, \"entry_type\": \"annotation\", \"annotation\": \"Your note\", \"changed_at\": \"...\"},\n  {\"id\": 2, \"entry_type\": \"change\", \"data\": {}, \"annotation\": null, \"changed_at\": \"...\"}\n]\n```\n\n")
		b.WriteString("`entry_type` is `\"annotation\"` for standalone notes and `\"change\"` for field-change events (which may also carry an `annotation` note).\n\n")

		fmt.Fprintf(b, "**Add annotation to a record** — returns 201 with the created entry.\n```\nPOST %s/{rowID}/history\nContent-Type: application/json\n\n{\"annotation\": \"Processed the request and set status to **Done**.\"}\n```\n\n", base)
		fmt.Fprintf(b, "**Update an annotation** (or add a note to a field-change entry)\n```\nPATCH %s/{rowID}/history/{histID}\nContent-Type: application/json\n\n{\"annotation\": \"Updated note text\"}\n```\n\n", base)
		fmt.Fprintf(b, "**Delete an annotation** — returns 204 No Content.\n```\nDELETE %s/{rowID}/history/{histID}\n```\n\n", base)
	}

	b.WriteString("### When to annotate\n\n")
	b.WriteString("- After creating or updating a record, add an annotation to document *why* or *what* was done\n")
	b.WriteString("- During refinement, use annotations to propose concepts or ask questions without modifying data\n")
	b.WriteString("- Patch a `\"change\"` entry's `histID` to add context to a specific field change\n\n")
}

func parseSelectOptions(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var opts []string
	if err := json.Unmarshal(raw, &opts); err == nil {
		return opts
	}
	var withChoices struct {
		Choices []string `json:"choices"`
	}
	if err := json.Unmarshal(raw, &withChoices); err == nil && len(withChoices.Choices) > 0 {
		return withChoices.Choices
	}
	var objOpts []struct {
		Value string `json:"value"`
		Label string `json:"label"`
	}
	if err := json.Unmarshal(raw, &objOpts); err == nil {
		out := make([]string, 0, len(objOpts))
		for _, o := range objOpts {
			if o.Label != "" {
				out = append(out, o.Label)
			} else {
				out = append(out, o.Value)
			}
		}
		return out
	}
	return nil
}
