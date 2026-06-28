package mcp

type toolManifest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func toolManifests() []toolManifest {
	str := func(desc string) map[string]any {
		return map[string]any{"type": "string", "description": desc}
	}
	strOpt := func(desc string) map[string]any {
		return map[string]any{"type": "string", "description": desc}
	}
	intProp := func(desc string) map[string]any {
		return map[string]any{"type": "integer", "description": desc}
	}
	obj := func(desc string) map[string]any {
		return map[string]any{"type": "object", "description": desc}
	}
	schema := func(props map[string]any, required ...string) map[string]any {
		s := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			s["required"] = required
		}
		return s
	}
	filterSchema := map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"col":   str("Column code"),
				"op":    str("Operator: is, is_not, contains, not_contains, eq, gt, lt, is_empty, is_not_empty, in, not_in"),
				"value": str("Filter value"),
			},
		},
		"description": "Row filters",
	}
	sortSchema := map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"col": str("Column code"),
				"dir": str("asc or desc"),
			},
		},
		"description": "Sort order",
	}
	columnTypes := "text, long-text, markdown, email, url, number, currency, percent, rating, date, datetime, checkbox, single-select, multi-select, file, image, symbol, emoji, created-at, updated-at, checklist, row-link, mermaid"

	return []toolManifest{
		{
			Name:        "get_workspace",
			Description: "Get workspace details including name, context, code, and a summary of all tables with their descriptions. Call this first to orient yourself.",
			InputSchema: schema(map[string]any{}),
		},
		{
			Name:        "list_tables",
			Description: "List all tables in the workspace.",
			InputSchema: schema(map[string]any{}),
		},
		{
			Name:        "create_table",
			Description: "Create a new table in the workspace.",
			InputSchema: schema(map[string]any{
				"name":    str("Table name"),
				"context": strOpt("Purpose description for agents"),
				"icon":    strOpt("Emoji icon"),
				"code":    strOpt("Short uppercase code (auto-generated if omitted)"),
			}, "name"),
		},
		{
			Name:        "update_table",
			Description: "Update a table's name, context, or icon.",
			InputSchema: schema(map[string]any{
				"table_code": str("Table code"),
				"name":       strOpt("New name"),
				"context":    strOpt("New context description"),
				"icon":       strOpt("New emoji icon"),
			}, "table_code"),
		},
		{
			Name:        "delete_table",
			Description: "Delete a table and all its rows.",
			InputSchema: schema(map[string]any{
				"table_code": str("Table code"),
			}, "table_code"),
		},
		{
			Name:        "list_columns",
			Description: "List columns for a table including their types and options.",
			InputSchema: schema(map[string]any{
				"table_code": str("Table code"),
			}, "table_code"),
		},
		{
			Name:        "create_column",
			Description: "Add a column to a table. Valid types: " + columnTypes,
			InputSchema: schema(map[string]any{
				"table_code": str("Table code"),
				"name":       str("Column name"),
				"type":       str("Column type"),
				"options":    obj("Column options (e.g. {\"choices\":[\"A\",\"B\"]} for select columns)"),
			}, "table_code", "name", "type"),
		},
		{
			Name:        "update_column",
			Description: "Rename a column or change its options.",
			InputSchema: schema(map[string]any{
				"table_code":  str("Table code"),
				"column_code": str("Column code"),
				"name":        strOpt("New name"),
				"type":        strOpt("New type"),
				"options":     obj("New options"),
			}, "table_code", "column_code"),
		},
		{
			Name:        "delete_column",
			Description: "Delete a column from a table.",
			InputSchema: schema(map[string]any{
				"table_code":  str("Table code"),
				"column_code": str("Column code"),
			}, "table_code", "column_code"),
		},
		{
			Name:        "query_rows",
			Description: "List rows with optional filters, sorts, search, and pagination.",
			InputSchema: schema(map[string]any{
				"table_code": str("Table code"),
				"filters":    filterSchema,
				"sorts":      sortSchema,
				"search":     strOpt("Full-text search string"),
				"page":       intProp("Page number (default 1)"),
				"page_size":  intProp("Rows per page (default 100, max 500)"),
			}, "table_code"),
		},
		{
			Name:        "get_row",
			Description: "Fetch a single row by ID.",
			InputSchema: schema(map[string]any{
				"table_code": str("Table code"),
				"row_id":     intProp("Row ID"),
			}, "table_code", "row_id"),
		},
		{
			Name:        "create_row",
			Description: "Insert a new row. Data keys are column codes.",
			InputSchema: schema(map[string]any{
				"table_code": str("Table code"),
				"data":       obj("Row data keyed by column code"),
			}, "table_code"),
		},
		{
			Name:        "patch_row",
			Description: "Partially update a row. Only supplied fields are changed.",
			InputSchema: schema(map[string]any{
				"table_code": str("Table code"),
				"row_id":     intProp("Row ID"),
				"data":       obj("Fields to update, keyed by column code"),
			}, "table_code", "row_id", "data"),
		},
		{
			Name:        "delete_row",
			Description: "Delete a row. Fails if other rows reference it via row-link columns.",
			InputSchema: schema(map[string]any{
				"table_code": str("Table code"),
				"row_id":     intProp("Row ID"),
			}, "table_code", "row_id"),
		},
		{
			Name:        "bulk_patch_rows",
			Description: "Apply a patch to all rows matching the given filters. Set preview=true to count without modifying.",
			InputSchema: schema(map[string]any{
				"table_code": str("Table code"),
				"filters":    filterSchema,
				"data":       obj("Fields to update, keyed by column code"),
				"preview":    map[string]any{"type": "boolean", "description": "If true, return count without modifying"},
			}, "table_code", "data"),
		},
		{
			Name:        "list_referencing_rows",
			Description: "Find rows in other tables that link to the given row via row-link columns.",
			InputSchema: schema(map[string]any{
				"table_code": str("Table code of the target row"),
				"row_id":     intProp("Target row ID"),
				"page":       intProp("Page number (default 1)"),
				"page_size":  intProp("Rows per page (default 50)"),
			}, "table_code", "row_id"),
		},
		{
			Name:        "get_row_history",
			Description: "Read the change history and annotations for a row.",
			InputSchema: schema(map[string]any{
				"table_code": str("Table code"),
				"row_id":     intProp("Row ID"),
			}, "table_code", "row_id"),
		},
		{
			Name:        "annotate_row",
			Description: "Add an annotation to a row's history explaining what was done and why.",
			InputSchema: schema(map[string]any{
				"table_code": str("Table code"),
				"row_id":     intProp("Row ID"),
				"text":       str("Annotation text (markdown supported)"),
			}, "table_code", "row_id", "text"),
		},
		{
			Name:        "list_scripts",
			Description: "List automation scripts in the workspace.",
			InputSchema: schema(map[string]any{}),
		},
		{
			Name:        "create_script",
			Description: "Create a Lua automation script. event_type: record.created, record.updated, or record.deleted.",
			InputSchema: schema(map[string]any{
				"name":        str("Script name"),
				"event_type":  str("Trigger: record.created, record.updated, record.deleted"),
				"code":        str("Lua 5.1 script code"),
				"enabled":     map[string]any{"type": "boolean", "description": "Enable on creation (default true)"},
				"table_codes": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Table codes to scope (omit for workspace-wide)"},
			}, "name", "event_type", "code"),
		},
		{
			Name:        "update_script",
			Description: "Update a script's code, name, event type, or enabled state.",
			InputSchema: schema(map[string]any{
				"script_id":   intProp("Script ID"),
				"name":        strOpt("New name"),
				"event_type":  strOpt("New event type"),
				"code":        strOpt("New Lua code"),
				"enabled":     map[string]any{"type": "boolean", "description": "Enable or disable"},
				"table_codes": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Table codes (null = keep existing, [] = workspace-wide)"},
			}, "script_id"),
		},
		{
			Name:        "delete_script",
			Description: "Delete a script.",
			InputSchema: schema(map[string]any{
				"script_id": intProp("Script ID"),
			}, "script_id"),
		},
		{
			Name:        "list_script_executions",
			Description: "Read recent execution logs for a script.",
			InputSchema: schema(map[string]any{
				"script_id": intProp("Script ID"),
				"limit":     intProp("Max results (default 50, max 200)"),
			}, "script_id"),
		},
	}
}
