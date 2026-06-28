package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/saintedlama/degubase/internal/automation"
	"github.com/saintedlama/degubase/internal/identity"
	"github.com/saintedlama/degubase/internal/infrastructure/store"
	"github.com/saintedlama/degubase/internal/models"
	"github.com/saintedlama/degubase/internal/records"
	dbschema "github.com/saintedlama/degubase/internal/schema"
)

// ── Test helpers ──────────────────────────────────────────────────────────────

type testEnv struct {
	h     *Handler
	ws    *models.Workspace
	ident *identity.SQLite
}

func newEnv(t *testing.T) *testEnv {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.New(fmt.Sprintf("file:%s?_pragma=foreign_keys(1)", dbPath))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	db := s.DB()
	t.Cleanup(func() { db.Close() })

	identStore := identity.NewSQLite(db)
	schemStore := dbschema.NewSQLite(db)
	rowStore := records.NewSQLite(db)
	autoStore := automation.NewSQLite(db)
	rowSvc := &records.RowService{Store: rowStore, Cols: schemStore}

	ws, err := identStore.CreateWorkspace(context.Background(), "Test Workspace", "for testing", "TEST")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	return &testEnv{
		h:     NewHandler(schemStore, rowStore, rowSvc, autoStore),
		ws:    ws,
		ident: identStore,
	}
}

func call(t *testing.T, h *Handler, ws *models.Workspace, method string, args any) toolResult {
	t.Helper()
	argsJSON, _ := json.Marshal(args)
	id := json.RawMessage(`1`)
	params, _ := json.Marshal(map[string]any{"name": method, "arguments": json.RawMessage(argsJSON)})
	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "tools/call",
		Params:  params,
	}
	resp := h.dispatch(context.Background(), ws, req)
	if resp.Error != nil {
		t.Fatalf("dispatch error for %s: %s", method, resp.Error.Message)
	}
	result, ok := resp.Result.(toolResult)
	if !ok {
		t.Fatalf("expected toolResult, got %T", resp.Result)
	}
	return result
}

func mustDecode[T any](t *testing.T, tr toolResult) T {
	t.Helper()
	if tr.IsError {
		t.Fatalf("tool returned error: %s", tr.Content[0].Text)
	}
	var v T
	if err := json.Unmarshal([]byte(tr.Content[0].Text), &v); err != nil {
		t.Fatalf("decode result: %v\ntext: %s", err, tr.Content[0].Text)
	}
	return v
}

// ── Protocol ──────────────────────────────────────────────────────────────────

func TestInitialize(t *testing.T) {
	env := newEnv(t)
	id := json.RawMessage(`1`)
	resp := env.h.dispatch(context.Background(), env.ws, &jsonRPCRequest{
		JSONRPC: "2.0", ID: &id, Method: "initialize",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	m, ok := resp.Result.(map[string]any)
	if !ok || m["protocolVersion"] == nil {
		t.Fatal("expected initialize result with protocolVersion")
	}
}

func TestToolsList(t *testing.T) {
	env := newEnv(t)
	id := json.RawMessage(`1`)
	resp := env.h.dispatch(context.Background(), env.ws, &jsonRPCRequest{
		JSONRPC: "2.0", ID: &id, Method: "tools/list",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	m := resp.Result.(map[string]any)
	tools := m["tools"].([]toolManifest)
	if len(tools) != 23 {
		t.Errorf("expected 23 tools, got %d", len(tools))
	}
}

func TestUnknownMethod(t *testing.T) {
	env := newEnv(t)
	id := json.RawMessage(`1`)
	resp := env.h.dispatch(context.Background(), env.ws, &jsonRPCRequest{
		JSONRPC: "2.0", ID: &id, Method: "bogus/method",
	})
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
}

// ── Workspace ─────────────────────────────────────────────────────────────────

func TestGetWorkspace(t *testing.T) {
	env := newEnv(t)
	result := call(t, env.h, env.ws, "get_workspace", map[string]any{})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}
	ws := mustDecode[models.Workspace](t, result)
	if ws.Code != env.ws.Code {
		t.Errorf("expected code %s, got %s", env.ws.Code, ws.Code)
	}
}

// ── Tables ────────────────────────────────────────────────────────────────────

func TestListTablesEmpty(t *testing.T) {
	env := newEnv(t)
	result := call(t, env.h, env.ws, "list_tables", map[string]any{})
	tables := mustDecode[[]models.Table](t, result)
	if len(tables) != 0 {
		t.Errorf("expected empty, got %d", len(tables))
	}
}

func TestCreateTable(t *testing.T) {
	env := newEnv(t)
	result := call(t, env.h, env.ws, "create_table", map[string]any{
		"name":    "Issues",
		"context": "Bug and feature tracking",
		"icon":    "🐛",
	})
	tbl := mustDecode[models.Table](t, result)
	if tbl.Name != "Issues" {
		t.Errorf("expected name Issues, got %s", tbl.Name)
	}
	if tbl.Code == "" {
		t.Error("expected non-empty code")
	}
}

func TestUpdateTable(t *testing.T) {
	env := newEnv(t)
	created := mustDecode[models.Table](t, call(t, env.h, env.ws, "create_table", map[string]any{"name": "Old"}))
	updated := mustDecode[models.Table](t, call(t, env.h, env.ws, "update_table", map[string]any{
		"table_code": created.Code,
		"name":       "New",
		"context":    "Updated context",
	}))
	if updated.Name != "New" {
		t.Errorf("expected New, got %s", updated.Name)
	}
}

func TestUpdateTableNotFound(t *testing.T) {
	env := newEnv(t)
	result := call(t, env.h, env.ws, "update_table", map[string]any{"table_code": "NOPE"})
	if !result.IsError {
		t.Fatal("expected error for missing table")
	}
}

func TestDeleteTable(t *testing.T) {
	env := newEnv(t)
	created := mustDecode[models.Table](t, call(t, env.h, env.ws, "create_table", map[string]any{"name": "Temp"}))
	result := call(t, env.h, env.ws, "delete_table", map[string]any{"table_code": created.Code})
	if result.IsError {
		t.Fatalf("delete failed: %s", result.Content[0].Text)
	}
	tables := mustDecode[[]models.Table](t, call(t, env.h, env.ws, "list_tables", map[string]any{}))
	if len(tables) != 0 {
		t.Errorf("expected 0 tables after delete, got %d", len(tables))
	}
}

// ── Columns ───────────────────────────────────────────────────────────────────

func TestCreateAndListColumns(t *testing.T) {
	env := newEnv(t)
	tbl := mustDecode[models.Table](t, call(t, env.h, env.ws, "create_table", map[string]any{"name": "T"}))

	call(t, env.h, env.ws, "create_column", map[string]any{
		"table_code": tbl.Code, "name": "Title", "type": "text",
	})
	call(t, env.h, env.ws, "create_column", map[string]any{
		"table_code": tbl.Code, "name": "Done", "type": "checkbox",
	})

	cols := mustDecode[[]models.Column](t, call(t, env.h, env.ws, "list_columns", map[string]any{"table_code": tbl.Code}))
	if len(cols) != 2 {
		t.Errorf("expected 2 columns, got %d", len(cols))
	}
}

func TestCreateColumnInvalidType(t *testing.T) {
	env := newEnv(t)
	tbl := mustDecode[models.Table](t, call(t, env.h, env.ws, "create_table", map[string]any{"name": "T"}))
	result := call(t, env.h, env.ws, "create_column", map[string]any{
		"table_code": tbl.Code, "name": "X", "type": "bogus-type",
	})
	if !result.IsError {
		t.Fatal("expected error for invalid column type")
	}
}

func TestUpdateColumn(t *testing.T) {
	env := newEnv(t)
	tbl := mustDecode[models.Table](t, call(t, env.h, env.ws, "create_table", map[string]any{"name": "T"}))
	col := mustDecode[models.Column](t, call(t, env.h, env.ws, "create_column", map[string]any{
		"table_code": tbl.Code, "name": "Old Name", "type": "text",
	}))
	newName := "New Name"
	updated := mustDecode[models.Column](t, call(t, env.h, env.ws, "update_column", map[string]any{
		"table_code":  tbl.Code,
		"column_code": col.Code,
		"name":        newName,
	}))
	if updated.Name != newName {
		t.Errorf("expected %s, got %s", newName, updated.Name)
	}
}

func TestDeleteColumn(t *testing.T) {
	env := newEnv(t)
	tbl := mustDecode[models.Table](t, call(t, env.h, env.ws, "create_table", map[string]any{"name": "T"}))
	col := mustDecode[models.Column](t, call(t, env.h, env.ws, "create_column", map[string]any{
		"table_code": tbl.Code, "name": "Temp", "type": "text",
	}))
	result := call(t, env.h, env.ws, "delete_column", map[string]any{
		"table_code": tbl.Code, "column_code": col.Code,
	})
	if result.IsError {
		t.Fatalf("delete column failed: %s", result.Content[0].Text)
	}
	cols := mustDecode[[]models.Column](t, call(t, env.h, env.ws, "list_columns", map[string]any{"table_code": tbl.Code}))
	if len(cols) != 0 {
		t.Errorf("expected 0 columns, got %d", len(cols))
	}
}

// ── Rows ──────────────────────────────────────────────────────────────────────

func setupTableWithColumn(t *testing.T, env *testEnv) models.Table {
	t.Helper()
	tbl := mustDecode[models.Table](t, call(t, env.h, env.ws, "create_table", map[string]any{"name": "Items"}))
	call(t, env.h, env.ws, "create_column", map[string]any{
		"table_code": tbl.Code, "name": "Name", "type": "text",
	})
	return tbl
}

func TestCreateAndQueryRows(t *testing.T) {
	env := newEnv(t)
	tbl := setupTableWithColumn(t, env)

	cols := mustDecode[[]models.Column](t, call(t, env.h, env.ws, "list_columns", map[string]any{"table_code": tbl.Code}))
	nameCol := cols[0]

	data, _ := json.Marshal(map[string]string{nameCol.Code: "Alice"})
	row := mustDecode[models.Row](t, call(t, env.h, env.ws, "create_row", map[string]any{
		"table_code": tbl.Code,
		"data":       json.RawMessage(data),
	}))
	if row.ID == 0 {
		t.Fatal("expected non-zero row ID")
	}

	result := call(t, env.h, env.ws, "query_rows", map[string]any{"table_code": tbl.Code})
	paged := mustDecode[models.PagedRows](t, result)
	if paged.Total != 1 {
		t.Errorf("expected total=1, got %d", paged.Total)
	}
}

func TestGetRow(t *testing.T) {
	env := newEnv(t)
	tbl := setupTableWithColumn(t, env)
	created := mustDecode[models.Row](t, call(t, env.h, env.ws, "create_row", map[string]any{"table_code": tbl.Code}))

	got := mustDecode[models.Row](t, call(t, env.h, env.ws, "get_row", map[string]any{
		"table_code": tbl.Code, "row_id": created.ID,
	}))
	if got.ID != created.ID {
		t.Errorf("expected id=%d, got %d", created.ID, got.ID)
	}
}

func TestGetRowNotFound(t *testing.T) {
	env := newEnv(t)
	tbl := mustDecode[models.Table](t, call(t, env.h, env.ws, "create_table", map[string]any{"name": "T"}))
	result := call(t, env.h, env.ws, "get_row", map[string]any{"table_code": tbl.Code, "row_id": 99999})
	if !result.IsError {
		t.Fatal("expected error for missing row")
	}
}

func TestPatchRow(t *testing.T) {
	env := newEnv(t)
	tbl := setupTableWithColumn(t, env)
	cols := mustDecode[[]models.Column](t, call(t, env.h, env.ws, "list_columns", map[string]any{"table_code": tbl.Code}))
	nameCol := cols[0]

	created := mustDecode[models.Row](t, call(t, env.h, env.ws, "create_row", map[string]any{"table_code": tbl.Code}))
	patch, _ := json.Marshal(map[string]string{nameCol.Code: "Updated"})
	patched := mustDecode[models.Row](t, call(t, env.h, env.ws, "patch_row", map[string]any{
		"table_code": tbl.Code,
		"row_id":     created.ID,
		"data":       json.RawMessage(patch),
	}))

	var data map[string]any
	json.Unmarshal(patched.Data, &data)
	if data[nameCol.Code] != "Updated" {
		t.Errorf("expected Updated, got %v", data[nameCol.Code])
	}
}

func TestDeleteRow(t *testing.T) {
	env := newEnv(t)
	tbl := setupTableWithColumn(t, env)
	created := mustDecode[models.Row](t, call(t, env.h, env.ws, "create_row", map[string]any{"table_code": tbl.Code}))

	result := call(t, env.h, env.ws, "delete_row", map[string]any{"table_code": tbl.Code, "row_id": created.ID})
	if result.IsError {
		t.Fatalf("delete failed: %s", result.Content[0].Text)
	}
	paged := mustDecode[models.PagedRows](t, call(t, env.h, env.ws, "query_rows", map[string]any{"table_code": tbl.Code}))
	if paged.Total != 0 {
		t.Errorf("expected 0 rows after delete, got %d", paged.Total)
	}
}

func TestBulkPatchRows(t *testing.T) {
	env := newEnv(t)
	tbl := setupTableWithColumn(t, env)
	cols := mustDecode[[]models.Column](t, call(t, env.h, env.ws, "list_columns", map[string]any{"table_code": tbl.Code}))
	nameCol := cols[0]

	for i := range 3 {
		d, _ := json.Marshal(map[string]string{nameCol.Code: fmt.Sprintf("Item %d", i)})
		call(t, env.h, env.ws, "create_row", map[string]any{"table_code": tbl.Code, "data": json.RawMessage(d)})
	}

	patch, _ := json.Marshal(map[string]string{nameCol.Code: "Patched"})
	result := call(t, env.h, env.ws, "bulk_patch_rows", map[string]any{
		"table_code": tbl.Code,
		"filters":    []any{},
		"data":       json.RawMessage(patch),
	})
	if result.IsError {
		t.Fatalf("bulk patch failed: %s", result.Content[0].Text)
	}
	var m map[string]any
	json.Unmarshal([]byte(result.Content[0].Text), &m)
	if m["count"].(float64) != 3 {
		t.Errorf("expected count=3, got %v", m["count"])
	}
}

func TestBulkPatchRowsPreview(t *testing.T) {
	env := newEnv(t)
	tbl := setupTableWithColumn(t, env)
	call(t, env.h, env.ws, "create_row", map[string]any{"table_code": tbl.Code})
	call(t, env.h, env.ws, "create_row", map[string]any{"table_code": tbl.Code})

	patch, _ := json.Marshal(map[string]string{})
	result := call(t, env.h, env.ws, "bulk_patch_rows", map[string]any{
		"table_code": tbl.Code,
		"filters":    []any{},
		"data":       json.RawMessage(patch),
		"preview":    true,
	})
	if result.IsError {
		t.Fatalf("preview failed: %s", result.Content[0].Text)
	}
	var m map[string]any
	json.Unmarshal([]byte(result.Content[0].Text), &m)
	if m["count"].(float64) != 2 {
		t.Errorf("expected preview count=2, got %v", m["count"])
	}
}

// ── Row history ───────────────────────────────────────────────────────────────

func TestGetRowHistory(t *testing.T) {
	env := newEnv(t)
	tbl := setupTableWithColumn(t, env)
	row := mustDecode[models.Row](t, call(t, env.h, env.ws, "create_row", map[string]any{"table_code": tbl.Code}))

	history := mustDecode[[]models.RowHistory](t, call(t, env.h, env.ws, "get_row_history", map[string]any{
		"table_code": tbl.Code, "row_id": row.ID,
	}))
	if len(history) < 1 {
		t.Errorf("expected at least 1 history entry, got %d", len(history))
	}
}

func TestAnnotateRow(t *testing.T) {
	env := newEnv(t)
	tbl := setupTableWithColumn(t, env)
	row := mustDecode[models.Row](t, call(t, env.h, env.ws, "create_row", map[string]any{"table_code": tbl.Code}))

	result := call(t, env.h, env.ws, "annotate_row", map[string]any{
		"table_code": tbl.Code,
		"row_id":     row.ID,
		"text":       "Added by the MCP agent",
	})
	entry := mustDecode[models.RowHistory](t, result)
	if entry.Annotation != "Added by the MCP agent" {
		t.Errorf("expected annotation text, got %q", entry.Annotation)
	}
}

// ── Linked records ────────────────────────────────────────────────────────────

func TestListReferencingRowsEmpty(t *testing.T) {
	env := newEnv(t)
	tbl := setupTableWithColumn(t, env)
	row := mustDecode[models.Row](t, call(t, env.h, env.ws, "create_row", map[string]any{"table_code": tbl.Code}))

	result := call(t, env.h, env.ws, "list_referencing_rows", map[string]any{
		"table_code": tbl.Code, "row_id": row.ID,
	})
	if result.IsError {
		t.Fatalf("list_referencing_rows failed: %s", result.Content[0].Text)
	}
}

// ── Scripts ───────────────────────────────────────────────────────────────────

func TestListScriptsEmpty(t *testing.T) {
	env := newEnv(t)
	result := call(t, env.h, env.ws, "list_scripts", map[string]any{})
	scripts := mustDecode[[]models.Script](t, result)
	if len(scripts) != 0 {
		t.Errorf("expected 0 scripts, got %d", len(scripts))
	}
}

func TestCreateScript(t *testing.T) {
	env := newEnv(t)
	result := call(t, env.h, env.ws, "create_script", map[string]any{
		"name":       "My Script",
		"event_type": "record.created",
		"code":       `log.info("row created")`,
	})
	script := mustDecode[models.Script](t, result)
	if script.Name != "My Script" {
		t.Errorf("expected My Script, got %s", script.Name)
	}
	if !script.Enabled {
		t.Error("expected enabled=true by default")
	}
}

func TestCreateScriptInvalidEventType(t *testing.T) {
	env := newEnv(t)
	result := call(t, env.h, env.ws, "create_script", map[string]any{
		"name": "X", "event_type": "bogus", "code": "log.info('x')",
	})
	if !result.IsError {
		t.Fatal("expected error for invalid event_type")
	}
}

func TestUpdateScript(t *testing.T) {
	env := newEnv(t)
	script := mustDecode[models.Script](t, call(t, env.h, env.ws, "create_script", map[string]any{
		"name": "Original", "event_type": "record.created", "code": "-- v1",
	}))
	newCode := "-- v2"
	enabled := false
	updated := mustDecode[models.Script](t, call(t, env.h, env.ws, "update_script", map[string]any{
		"script_id": script.ID,
		"code":      newCode,
		"enabled":   enabled,
	}))
	if updated.Code != newCode {
		t.Errorf("expected %q, got %q", newCode, updated.Code)
	}
	if updated.Enabled {
		t.Error("expected enabled=false")
	}
}

func TestDeleteScript(t *testing.T) {
	env := newEnv(t)
	script := mustDecode[models.Script](t, call(t, env.h, env.ws, "create_script", map[string]any{
		"name": "Temp", "event_type": "record.deleted", "code": "-- bye",
	}))
	result := call(t, env.h, env.ws, "delete_script", map[string]any{"script_id": script.ID})
	if result.IsError {
		t.Fatalf("delete failed: %s", result.Content[0].Text)
	}
	scripts := mustDecode[[]models.Script](t, call(t, env.h, env.ws, "list_scripts", map[string]any{}))
	if len(scripts) != 0 {
		t.Errorf("expected 0 scripts, got %d", len(scripts))
	}
}

func TestListScriptExecutions(t *testing.T) {
	env := newEnv(t)
	script := mustDecode[models.Script](t, call(t, env.h, env.ws, "create_script", map[string]any{
		"name": "S", "event_type": "record.created", "code": "-- noop",
	}))
	result := call(t, env.h, env.ws, "list_script_executions", map[string]any{"script_id": script.ID})
	execs := mustDecode[[]models.ScriptExecution](t, result)
	if len(execs) != 0 {
		t.Errorf("expected 0 executions, got %d", len(execs))
	}
}

func TestScriptIsolatedByWorkspace(t *testing.T) {
	env := newEnv(t)
	// Create a second workspace in the same database — IDs will differ.
	ws2, err := env.ident.CreateWorkspace(context.Background(), "Other", "", "OTH")
	if err != nil {
		t.Fatalf("create second workspace: %v", err)
	}

	script := mustDecode[models.Script](t, call(t, env.h, env.ws, "create_script", map[string]any{
		"name": "Secret", "event_type": "record.created", "code": "-- secret",
	}))
	// ws2 does not own the script; the handler must reject the delete.
	result := call(t, env.h, ws2, "delete_script", map[string]any{"script_id": script.ID})
	if !result.IsError {
		t.Fatal("expected error when deleting script from wrong workspace")
	}
}

// ── Unknown tool ──────────────────────────────────────────────────────────────

func TestUnknownTool(t *testing.T) {
	env := newEnv(t)
	result := call(t, env.h, env.ws, "nonexistent_tool", map[string]any{})
	if !result.IsError {
		t.Fatal("expected error for unknown tool")
	}
}
