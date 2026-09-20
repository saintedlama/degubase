package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/saintedlama/degubase/internal/models"
)

func (h *Handler) dispatch(ctx context.Context, ws *models.Workspace, req *jsonRPCRequest) jsonRPCResponse {
	resp := jsonRPCResponse{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "degubase", "version": "1.0"},
		}
	case "ping":
		resp.Result = map[string]any{}
	case "tools/list":
		resp.Result = map[string]any{"tools": toolManifests()}
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(safeArgs(req.Params), &p); err != nil {
			resp.Error = &jsonRPCError{Code: -32600, Message: "invalid params"}
			return resp
		}
		resp.Result = h.callTool(ctx, ws, p.Name, safeArgs(p.Arguments))
	default:
		resp.Error = &jsonRPCError{Code: -32601, Message: "method not found: " + req.Method}
	}
	return resp
}

func (h *Handler) callTool(ctx context.Context, ws *models.Workspace, name string, args json.RawMessage) toolResult {
	switch name {
	case "get_workspace":
		return h.toolGetWorkspace(ctx, ws, args)
	case "list_tables":
		return h.toolListTables(ctx, ws, args)
	case "create_table":
		return h.toolCreateTable(ctx, ws, args)
	case "update_table":
		return h.toolUpdateTable(ctx, ws, args)
	case "delete_table":
		return h.toolDeleteTable(ctx, ws, args)
	case "list_columns":
		return h.toolListColumns(ctx, ws, args)
	case "create_column":
		return h.toolCreateColumn(ctx, ws, args)
	case "update_column":
		return h.toolUpdateColumn(ctx, ws, args)
	case "delete_column":
		return h.toolDeleteColumn(ctx, ws, args)
	case "query_rows":
		return h.toolQueryRows(ctx, ws, args)
	case "get_row":
		return h.toolGetRow(ctx, ws, args)
	case "create_row":
		return h.toolCreateRow(ctx, ws, args)
	case "patch_row":
		return h.toolPatchRow(ctx, ws, args)
	case "delete_row":
		return h.toolDeleteRow(ctx, ws, args)
	case "bulk_patch_rows":
		return h.toolBulkPatchRows(ctx, ws, args)
	case "list_referencing_rows":
		return h.toolListReferencingRows(ctx, ws, args)
	case "get_row_history":
		return h.toolGetRowHistory(ctx, ws, args)
	case "annotate_row":
		return h.toolAnnotateRow(ctx, ws, args)
	case "list_scripts":
		return h.toolListScripts(ctx, ws, args)
	case "create_script":
		return h.toolCreateScript(ctx, ws, args)
	case "update_script":
		return h.toolUpdateScript(ctx, ws, args)
	case "delete_script":
		return h.toolDeleteScript(ctx, ws, args)
	case "list_script_executions":
		return h.toolListScriptExecutions(ctx, ws, args)
	default:
		return errResult(fmt.Sprintf("unknown tool: %s", name))
	}
}

// ── Workspace ─────────────────────────────────────────────────────────────────

func (h *Handler) toolGetWorkspace(ctx context.Context, ws *models.Workspace, _ json.RawMessage) toolResult {
	tables, err := h.Schema.ListTables(ctx, ws.ID)
	if err != nil {
		return errResult("list tables: " + err.Error())
	}
	type tableOverview struct {
		Code    string `json:"code"`
		Name    string `json:"name"`
		Context string `json:"context,omitempty"`
		Icon    string `json:"icon,omitempty"`
	}
	overviews := make([]tableOverview, len(tables))
	for i, t := range tables {
		overviews[i] = tableOverview{Code: t.Code, Name: t.Name, Context: t.Context, Icon: t.Icon}
	}
	return textResult(map[string]any{
		"code":        ws.Code,
		"name":        ws.Name,
		"context":     ws.Context,
		"mcp_enabled": ws.MCPEnabled,
		"tables":      overviews,
	})
}

// ── Tables ────────────────────────────────────────────────────────────────────

func (h *Handler) toolListTables(ctx context.Context, ws *models.Workspace, _ json.RawMessage) toolResult {
	tables, err := h.Schema.ListTables(ctx, ws.ID)
	if err != nil {
		return errResult("list tables: " + err.Error())
	}
	if tables == nil {
		tables = []models.Table{}
	}
	return textResult(tables)
}

type createTableArgs struct {
	Name    string `json:"name"`
	Context string `json:"context"`
	Icon    string `json:"icon"`
	Code    string `json:"code"`
}

func (h *Handler) toolCreateTable(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a createTableArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	if a.Name == "" {
		return errResult("name is required")
	}
	t, err := h.Schema.CreateTable(ctx, ws.ID, a.Name, a.Context, a.Icon, a.Code)
	if err != nil {
		return errResult("create table: " + err.Error())
	}
	return textResult(t)
}

type updateTableArgs struct {
	TableCode string  `json:"table_code"`
	Name      *string `json:"name"`
	Context   *string `json:"context"`
	Icon      *string `json:"icon"`
}

func (h *Handler) toolUpdateTable(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a updateTableArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	t, err := h.Schema.GetTableByCode(ctx, ws.ID, a.TableCode)
	if err != nil {
		return errResult("get table: " + err.Error())
	}
	if t == nil {
		return errResult("table not found: " + a.TableCode)
	}
	name, tctx, icon := t.Name, t.Context, t.Icon
	if a.Name != nil {
		name = *a.Name
	}
	if a.Context != nil {
		tctx = *a.Context
	}
	if a.Icon != nil {
		icon = *a.Icon
	}
	updated, err := h.Schema.UpdateTable(ctx, t.ID, name, tctx, icon, t.DefaultViewID)
	if err != nil {
		return errResult("update table: " + err.Error())
	}
	return textResult(updated)
}

type tableCodeArgs struct {
	TableCode string `json:"table_code"`
}

func (h *Handler) toolDeleteTable(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a tableCodeArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	t, err := h.Schema.GetTableByCode(ctx, ws.ID, a.TableCode)
	if err != nil {
		return errResult("get table: " + err.Error())
	}
	if t == nil {
		return errResult("table not found: " + a.TableCode)
	}
	if err := h.Schema.DeleteTable(ctx, t.ID); err != nil {
		return errResult("delete table: " + err.Error())
	}
	return textResult(map[string]bool{"deleted": true})
}

// ── Columns ───────────────────────────────────────────────────────────────────

func (h *Handler) toolListColumns(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a tableCodeArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	t, err := h.Schema.GetTableByCode(ctx, ws.ID, a.TableCode)
	if err != nil {
		return errResult("get table: " + err.Error())
	}
	if t == nil {
		return errResult("table not found: " + a.TableCode)
	}
	cols, err := h.Schema.ListColumns(ctx, t.ID)
	if err != nil {
		return errResult("list columns: " + err.Error())
	}
	if cols == nil {
		cols = []models.Column{}
	}
	return textResult(cols)
}

type createColumnArgs struct {
	TableCode string          `json:"table_code"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Options   json.RawMessage `json:"options"`
}

func (h *Handler) toolCreateColumn(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a createColumnArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	if a.Name == "" {
		return errResult("name is required")
	}
	colType := models.ColumnType(a.Type)
	if !colType.Valid() {
		return errResult("invalid column type: " + a.Type)
	}
	t, err := h.Schema.GetTableByCode(ctx, ws.ID, a.TableCode)
	if err != nil {
		return errResult("get table: " + err.Error())
	}
	if t == nil {
		return errResult("table not found: " + a.TableCode)
	}
	col, err := h.Schema.CreateColumn(ctx, t.ID, a.Name, colType, a.Options)
	if err != nil {
		return errResult("create column: " + err.Error())
	}
	return textResult(col)
}

type updateColumnArgs struct {
	TableCode  string          `json:"table_code"`
	ColumnCode string          `json:"column_code"`
	Name       *string         `json:"name"`
	Type       *string         `json:"type"`
	Options    json.RawMessage `json:"options"`
}

func (h *Handler) toolUpdateColumn(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a updateColumnArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	t, err := h.Schema.GetTableByCode(ctx, ws.ID, a.TableCode)
	if err != nil || t == nil {
		return errResult("table not found: " + a.TableCode)
	}
	col, err := h.Schema.GetColumnByCode(ctx, t.ID, a.ColumnCode)
	if err != nil || col == nil {
		return errResult("column not found: " + a.ColumnCode)
	}
	name, colType, options := col.Name, col.Type, col.Options
	if a.Name != nil {
		name = *a.Name
	}
	if a.Type != nil {
		colType = models.ColumnType(*a.Type)
	}
	if len(a.Options) > 0 {
		options = a.Options
	}
	updated, err := h.Schema.UpdateColumn(ctx, col.ID, name, colType, options)
	if err != nil {
		return errResult("update column: " + err.Error())
	}
	return textResult(updated)
}

type columnArgs struct {
	TableCode  string `json:"table_code"`
	ColumnCode string `json:"column_code"`
}

func (h *Handler) toolDeleteColumn(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a columnArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	t, err := h.Schema.GetTableByCode(ctx, ws.ID, a.TableCode)
	if err != nil || t == nil {
		return errResult("table not found: " + a.TableCode)
	}
	col, err := h.Schema.GetColumnByCode(ctx, t.ID, a.ColumnCode)
	if err != nil || col == nil {
		return errResult("column not found: " + a.ColumnCode)
	}
	if err := h.Schema.DeleteColumn(ctx, col.ID); err != nil {
		return errResult("delete column: " + err.Error())
	}
	return textResult(map[string]bool{"deleted": true})
}

// ── Rows ──────────────────────────────────────────────────────────────────────

type queryRowsArgs struct {
	TableCode string             `json:"table_code"`
	Filters   []models.RowFilter `json:"filters"`
	Sorts     []models.RowSort   `json:"sorts"`
	Search    string             `json:"search"`
	Page      int                `json:"page"`
	PageSize  int                `json:"page_size"`
}

func (h *Handler) toolQueryRows(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a queryRowsArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	t, err := h.Schema.GetTableByCode(ctx, ws.ID, a.TableCode)
	if err != nil || t == nil {
		return errResult("table not found: " + a.TableCode)
	}
	page, pageSize := a.Page, a.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 100
	}
	if pageSize > 500 {
		pageSize = 500
	}
	rows, total, err := h.RowSvc.ListRows(ctx, t.ID, pageSize, (page-1)*pageSize, a.Filters, a.Sorts, a.Search)
	if err != nil {
		return errResult("query rows: " + err.Error())
	}
	if rows == nil {
		rows = []models.Row{}
	}
	return textResult(models.NewPagedResult(rows, total, page, pageSize))
}

type rowArgs struct {
	TableCode string `json:"table_code"`
	RowID     int64  `json:"row_id"`
}

func (h *Handler) toolGetRow(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a rowArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	t, err := h.Schema.GetTableByCode(ctx, ws.ID, a.TableCode)
	if err != nil || t == nil {
		return errResult("table not found: " + a.TableCode)
	}
	row, err := h.RowSvc.GetRow(ctx, t.ID, a.RowID)
	if err != nil {
		return errResult("get row: " + err.Error())
	}
	if row == nil {
		return errResult(fmt.Sprintf("row not found: %d", a.RowID))
	}
	return textResult(row)
}

type createRowArgs struct {
	TableCode string          `json:"table_code"`
	Data      json.RawMessage `json:"data"`
}

func (h *Handler) toolCreateRow(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a createRowArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	t, err := h.Schema.GetTableByCode(ctx, ws.ID, a.TableCode)
	if err != nil || t == nil {
		return errResult("table not found: " + a.TableCode)
	}
	data := a.Data
	if len(data) == 0 {
		data = json.RawMessage("{}")
	}
	row, err := h.RowSvc.CreateRow(ctx, t.ID, data)
	if err != nil {
		return errResult("create row: " + err.Error())
	}
	return textResult(row)
}

type patchRowArgs struct {
	TableCode string          `json:"table_code"`
	RowID     int64           `json:"row_id"`
	Data      json.RawMessage `json:"data"`
}

func (h *Handler) toolPatchRow(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a patchRowArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	t, err := h.Schema.GetTableByCode(ctx, ws.ID, a.TableCode)
	if err != nil || t == nil {
		return errResult("table not found: " + a.TableCode)
	}
	row, _, err := h.RowSvc.PatchRow(ctx, t.ID, a.RowID, a.Data, "")
	if err != nil {
		return errResult("patch row: " + err.Error())
	}
	return textResult(row)
}

func (h *Handler) toolDeleteRow(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a rowArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	t, err := h.Schema.GetTableByCode(ctx, ws.ID, a.TableCode)
	if err != nil || t == nil {
		return errResult("table not found: " + a.TableCode)
	}
	row, err := h.RowSvc.GetRow(ctx, t.ID, a.RowID)
	if err != nil {
		return errResult("get row: " + err.Error())
	}
	if row == nil {
		return errResult(fmt.Sprintf("row not found: %d", a.RowID))
	}
	refs, err := h.Rows.FindRowLinkReferences(ctx, ws.ID, a.RowID)
	if err != nil {
		return errResult("check references: " + err.Error())
	}
	if len(refs) > 0 {
		return errResult(fmt.Sprintf("row is referenced by %d other row(s) and cannot be deleted", len(refs)))
	}
	if err := h.RowSvc.DeleteRow(ctx, t.ID, a.RowID); err != nil {
		return errResult("delete row: " + err.Error())
	}
	return textResult(map[string]bool{"deleted": true})
}

type bulkPatchArgs struct {
	TableCode string             `json:"table_code"`
	Filters   []models.RowFilter `json:"filters"`
	Data      json.RawMessage    `json:"data"`
	Preview   bool               `json:"preview"`
}

func (h *Handler) toolBulkPatchRows(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a bulkPatchArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	t, err := h.Schema.GetTableByCode(ctx, ws.ID, a.TableCode)
	if err != nil || t == nil {
		return errResult("table not found: " + a.TableCode)
	}
	rows, count, err := h.RowSvc.BulkPatch(ctx, t.ID, a.Filters, a.Data, a.Preview)
	if err != nil {
		return errResult("bulk patch: " + err.Error())
	}
	if rows == nil {
		rows = []*models.Row{}
	}
	return textResult(map[string]any{"count": count, "rows": rows, "preview": a.Preview})
}

// ── Linked records ────────────────────────────────────────────────────────────

type referencingRowsArgs struct {
	TableCode string `json:"table_code"`
	RowID     int64  `json:"row_id"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
}

func (h *Handler) toolListReferencingRows(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a referencingRowsArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	page, pageSize := a.Page, a.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	refs, total, err := h.Rows.ListReferencingRows(ctx, ws.ID, a.RowID, pageSize, (page-1)*pageSize)
	if err != nil {
		return errResult("list referencing rows: " + err.Error())
	}
	type refRow struct {
		SourceRowID     int64  `json:"source_row_id"`
		SourceTableCode string `json:"source_table_code"`
		SourceTableName string `json:"source_table_name"`
		ViaColumnCode   string `json:"via_column_code"`
		ViaColumnName   string `json:"via_column_name"`
	}
	out := make([]refRow, len(refs))
	for i, r := range refs {
		out[i] = refRow{
			SourceRowID:     r.SourceRowID,
			SourceTableCode: r.SourceTableCode,
			SourceTableName: r.SourceTableName,
			ViaColumnCode:   r.ViaColumnCode,
			ViaColumnName:   r.ViaColumnName,
		}
	}
	return textResult(models.NewPagedResult(out, total, page, pageSize))
}

// ── Row history ───────────────────────────────────────────────────────────────

func (h *Handler) toolGetRowHistory(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a rowArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	t, err := h.Schema.GetTableByCode(ctx, ws.ID, a.TableCode)
	if err != nil || t == nil {
		return errResult("table not found: " + a.TableCode)
	}
	history, err := h.RowSvc.ListRowHistory(ctx, t.ID, a.RowID)
	if err != nil {
		return errResult("get row history: " + err.Error())
	}
	if history == nil {
		history = []models.RowHistory{}
	}
	return textResult(history)
}

type annotateRowArgs struct {
	TableCode string `json:"table_code"`
	RowID     int64  `json:"row_id"`
	Text      string `json:"text"`
}

func (h *Handler) toolAnnotateRow(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a annotateRowArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	if a.Text == "" {
		return errResult("text is required")
	}
	t, err := h.Schema.GetTableByCode(ctx, ws.ID, a.TableCode)
	if err != nil || t == nil {
		return errResult("table not found: " + a.TableCode)
	}
	entry, err := h.RowSvc.CreateAnnotation(ctx, t.ID, a.RowID, a.Text)
	if err != nil {
		return errResult("annotate row: " + err.Error())
	}
	return textResult(entry)
}

// ── Scripts ───────────────────────────────────────────────────────────────────

func (h *Handler) toolListScripts(ctx context.Context, ws *models.Workspace, _ json.RawMessage) toolResult {
	scripts, err := h.Auto.ListScripts(ctx, ws.ID)
	if err != nil {
		return errResult("list scripts: " + err.Error())
	}
	if scripts == nil {
		scripts = []models.Script{}
	}
	return textResult(scripts)
}

type createScriptArgs struct {
	Name       string   `json:"name"`
	EventType  string   `json:"event_type"`
	Code       string   `json:"code"`
	Enabled    *bool    `json:"enabled"`
	TableCodes []string `json:"table_codes"`
}

func (h *Handler) toolCreateScript(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a createScriptArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	if a.Name == "" || a.EventType == "" || a.Code == "" {
		return errResult("name, event_type, and code are required")
	}
	evtType := models.ScriptEventType(a.EventType)
	if !evtType.Valid() {
		return errResult("invalid event_type: " + a.EventType)
	}
	tableIDs, err := h.resolveTableCodes(ctx, ws.ID, a.TableCodes)
	if err != nil {
		return errResult(err.Error())
	}
	enabled := true
	if a.Enabled != nil {
		enabled = *a.Enabled
	}
	script, err := h.Auto.CreateScript(ctx, ws.ID, tableIDs, a.Name, evtType, a.Code, enabled)
	if err != nil {
		return errResult("create script: " + err.Error())
	}
	return textResult(script)
}

type updateScriptArgs struct {
	ScriptID   int64    `json:"script_id"`
	Name       *string  `json:"name"`
	EventType  *string  `json:"event_type"`
	Code       *string  `json:"code"`
	Enabled    *bool    `json:"enabled"`
	TableCodes []string `json:"table_codes"`
}

func (h *Handler) toolUpdateScript(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a updateScriptArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	script, err := h.Auto.GetScript(ctx, a.ScriptID)
	if err != nil || script == nil {
		return errResult(fmt.Sprintf("script not found: %d", a.ScriptID))
	}
	if script.WorkspaceID != ws.ID {
		return errResult(fmt.Sprintf("script not found: %d", a.ScriptID))
	}
	name, evtType, code, enabled := script.Name, script.EventType, script.Code, script.Enabled
	if a.Name != nil {
		name = *a.Name
	}
	if a.EventType != nil {
		evtType = models.ScriptEventType(*a.EventType)
	}
	if a.Code != nil {
		code = *a.Code
	}
	if a.Enabled != nil {
		enabled = *a.Enabled
	}
	tableIDs := script.TableIDs
	if a.TableCodes != nil {
		tableIDs, err = h.resolveTableCodes(ctx, ws.ID, a.TableCodes)
		if err != nil {
			return errResult(err.Error())
		}
	}
	updated, err := h.Auto.UpdateScript(ctx, a.ScriptID, tableIDs, name, evtType, code, enabled)
	if err != nil {
		return errResult("update script: " + err.Error())
	}
	return textResult(updated)
}

type scriptIDArgs struct {
	ScriptID int64 `json:"script_id"`
}

func (h *Handler) toolDeleteScript(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a scriptIDArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	script, err := h.Auto.GetScript(ctx, a.ScriptID)
	if err != nil || script == nil || script.WorkspaceID != ws.ID {
		return errResult(fmt.Sprintf("script not found: %d", a.ScriptID))
	}
	if err := h.Auto.DeleteScript(ctx, a.ScriptID); err != nil {
		return errResult("delete script: " + err.Error())
	}
	return textResult(map[string]bool{"deleted": true})
}

type listScriptExecutionsArgs struct {
	ScriptID int64 `json:"script_id"`
	Limit    int   `json:"limit"`
}

func (h *Handler) toolListScriptExecutions(ctx context.Context, ws *models.Workspace, args json.RawMessage) toolResult {
	var a listScriptExecutionsArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errResult("invalid arguments: " + err.Error())
	}
	script, err := h.Auto.GetScript(ctx, a.ScriptID)
	if err != nil || script == nil || script.WorkspaceID != ws.ID {
		return errResult(fmt.Sprintf("script not found: %d", a.ScriptID))
	}
	limit := a.Limit
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	execs, err := h.Auto.ListScriptExecutions(ctx, a.ScriptID, limit)
	if err != nil {
		return errResult("list script executions: " + err.Error())
	}
	if execs == nil {
		execs = []models.ScriptExecution{}
	}
	return textResult(execs)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (h *Handler) resolveTableCodes(ctx context.Context, wsID int64, codes []string) ([]int64, error) {
	ids := make([]int64, 0, len(codes))
	for _, code := range codes {
		t, err := h.Schema.GetTableByCode(ctx, wsID, code)
		if err != nil || t == nil {
			return nil, fmt.Errorf("table not found: %s", code)
		}
		ids = append(ids, t.ID)
	}
	return ids, nil
}
