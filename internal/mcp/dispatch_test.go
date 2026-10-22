package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/saintedlama/degubase/internal/models"
)

func TestDispatchPing(t *testing.T) {
	env := newEnv(t)
	id := json.RawMessage(`1`)
	resp := env.h.dispatch(context.Background(), env.ws, &jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "ping",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error on ping: %v", resp.Error)
	}
	m, ok := resp.Result.(map[string]any)
	if !ok || len(m) != 0 {
		t.Errorf("expected empty map for ping, got %v", resp.Result)
	}
}

func TestDispatchToolsCallInvalidParams(t *testing.T) {
	env := newEnv(t)
	id := json.RawMessage(`1`)
	resp := env.h.dispatch(context.Background(), env.ws, &jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "tools/call",
		Params:  json.RawMessage(`not-a-valid-json`),
	})
	if resp.Error == nil {
		t.Fatal("expected error for invalid tools/call params")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("expected code -32600, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "invalid params" {
		t.Errorf("expected 'invalid params', got %s", resp.Error.Message)
	}
}

func TestCallToolInvalidArguments(t *testing.T) {
	env := newEnv(t)
	ctx := context.Background()

	tools := []string{
		"create_table", "update_table", "delete_table",
		"list_columns", "create_column", "update_column", "delete_column",
		"query_rows", "get_row", "create_row", "patch_row", "delete_row",
		"bulk_patch_rows", "list_referencing_rows", "get_row_history",
		"annotate_row", "create_script", "update_script", "delete_script",
		"list_script_executions",
	}

	for _, name := range tools {
		t.Run(name, func(t *testing.T) {
			res := env.h.callTool(ctx, env.ws, name, json.RawMessage(`{invalid-json`))
			if !res.IsError {
				t.Fatalf("expected error for invalid arguments in %s", name)
			}
			if len(res.Content) == 0 || !strings.Contains(res.Content[0].Text, "invalid arguments") {
				t.Errorf("expected error message to mention 'invalid arguments', got %v", res.Content)
			}
		})
	}
}

func TestGetWorkspaceDetailed(t *testing.T) {
	env := newEnv(t)
	call(t, env.h, env.ws, "create_table", map[string]any{
		"name":    "Detailed Table",
		"context": "Has details",
		"icon":    "📋",
		"code":    "DET",
	})

	result := call(t, env.h, env.ws, "get_workspace", map[string]any{})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].Text), &m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if m["code"] != env.ws.Code {
		t.Errorf("expected ws code %s, got %v", env.ws.Code, m["code"])
	}
	tables, ok := m["tables"].([]any)
	if !ok || len(tables) != 1 {
		t.Fatalf("expected 1 table in workspace overview, got %v", m["tables"])
	}
	tbl := tables[0].(map[string]any)
	if tbl["name"] != "Detailed Table" || tbl["icon"] != "📋" || tbl["context"] != "Has details" {
		t.Errorf("unexpected table details: %v", tbl)
	}
}

func TestCreateTableValidation(t *testing.T) {
	env := newEnv(t)
	res := call(t, env.h, env.ws, "create_table", map[string]any{"name": ""})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "name is required") {
		t.Fatalf("expected 'name is required', got: %v", res)
	}

	res = call(t, env.h, env.ws, "create_table", map[string]any{
		"name": "Custom Code",
		"code": "MY_CODE",
	})
	tbl := mustDecode[models.Table](t, res)
	if tbl.Code != "MY_CODE" {
		t.Errorf("expected MY_CODE, got %s", tbl.Code)
	}
}

func TestUpdateTableFields(t *testing.T) {
	env := newEnv(t)
	created := mustDecode[models.Table](t, call(t, env.h, env.ws, "create_table", map[string]any{"name": "Initial"}))

	newCtx := "Updated ctx"
	newIcon := "⭐"
	updated := mustDecode[models.Table](t, call(t, env.h, env.ws, "update_table", map[string]any{
		"table_code": created.Code,
		"context":    &newCtx,
		"icon":       &newIcon,
	}))
	if updated.Name != "Initial" {
		t.Errorf("expected unchanged name 'Initial', got %s", updated.Name)
	}
	if updated.Context != newCtx {
		t.Errorf("expected context %s, got %s", newCtx, updated.Context)
	}
	if updated.Icon != newIcon {
		t.Errorf("expected icon %s, got %s", newIcon, updated.Icon)
	}
}

func TestDeleteTableNotFound(t *testing.T) {
	env := newEnv(t)
	res := call(t, env.h, env.ws, "delete_table", map[string]any{"table_code": "NONEXISTENT"})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "table not found") {
		t.Fatalf("expected 'table not found' error, got %v", res)
	}
}

func TestListColumnsNotFound(t *testing.T) {
	env := newEnv(t)
	res := call(t, env.h, env.ws, "list_columns", map[string]any{"table_code": "NONEXISTENT"})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "table not found") {
		t.Fatalf("expected 'table not found' error, got %v", res)
	}
}

func TestCreateColumnValidation(t *testing.T) {
	env := newEnv(t)
	tbl := setupTableWithColumn(t, env)

	res := call(t, env.h, env.ws, "create_column", map[string]any{
		"table_code": tbl.Code,
		"name":       "",
		"type":       "text",
	})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "name is required") {
		t.Fatalf("expected 'name is required', got: %v", res)
	}

	res = call(t, env.h, env.ws, "create_column", map[string]any{
		"table_code": "NONEXISTENT",
		"name":       "Col",
		"type":       "text",
	})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "table not found") {
		t.Fatalf("expected 'table not found', got %v", res)
	}
}

func TestUpdateColumnValidation(t *testing.T) {
	env := newEnv(t)
	tbl := setupTableWithColumn(t, env)
	cols := mustDecode[[]models.Column](t, call(t, env.h, env.ws, "list_columns", map[string]any{"table_code": tbl.Code}))
	col := cols[0]

	res := call(t, env.h, env.ws, "update_column", map[string]any{
		"table_code":  "NONEXISTENT",
		"column_code": col.Code,
	})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "table not found") {
		t.Fatalf("expected 'table not found', got %v", res)
	}

	res = call(t, env.h, env.ws, "update_column", map[string]any{
		"table_code":  tbl.Code,
		"column_code": "NONEXISTENT",
	})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "column not found") {
		t.Fatalf("expected 'column not found', got %v", res)
	}

	newType := "number"
	opts := json.RawMessage(`{"decimal_places": 2}`)
	updated := mustDecode[models.Column](t, call(t, env.h, env.ws, "update_column", map[string]any{
		"table_code":  tbl.Code,
		"column_code": col.Code,
		"type":        &newType,
		"options":     opts,
	}))
	if string(updated.Type) != newType {
		t.Errorf("expected type %s, got %s", newType, updated.Type)
	}
}

func TestDeleteColumnValidation(t *testing.T) {
	env := newEnv(t)
	tbl := setupTableWithColumn(t, env)
	cols := mustDecode[[]models.Column](t, call(t, env.h, env.ws, "list_columns", map[string]any{"table_code": tbl.Code}))
	col := cols[0]

	res := call(t, env.h, env.ws, "delete_column", map[string]any{
		"table_code":  "NONEXISTENT",
		"column_code": col.Code,
	})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "table not found") {
		t.Fatalf("expected 'table not found', got %v", res)
	}

	res = call(t, env.h, env.ws, "delete_column", map[string]any{
		"table_code":  tbl.Code,
		"column_code": "NONEXISTENT",
	})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "column not found") {
		t.Fatalf("expected 'column not found', got %v", res)
	}
}

func TestQueryRowsValidationAndBoundaries(t *testing.T) {
	env := newEnv(t)
	tbl := setupTableWithColumn(t, env)
	cols := mustDecode[[]models.Column](t, call(t, env.h, env.ws, "list_columns", map[string]any{"table_code": tbl.Code}))
	nameCol := cols[0]

	res := call(t, env.h, env.ws, "query_rows", map[string]any{"table_code": "NONEXISTENT"})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "table not found") {
		t.Fatalf("expected 'table not found', got %v", res)
	}

	d1, _ := json.Marshal(map[string]string{nameCol.Code: "Alpha"})
	d2, _ := json.Marshal(map[string]string{nameCol.Code: "Beta"})
	call(t, env.h, env.ws, "create_row", map[string]any{"table_code": tbl.Code, "data": json.RawMessage(d1)})
	call(t, env.h, env.ws, "create_row", map[string]any{"table_code": tbl.Code, "data": json.RawMessage(d2)})

	res = call(t, env.h, env.ws, "query_rows", map[string]any{
		"table_code": tbl.Code,
		"page":       0,
		"page_size":  0,
	})
	paged := mustDecode[models.PagedRows](t, res)
	if paged.Page != 1 || paged.PageSize != 100 {
		t.Errorf("expected page=1, pageSize=100, got page=%d, pageSize=%d", paged.Page, paged.PageSize)
	}

	res = call(t, env.h, env.ws, "query_rows", map[string]any{
		"table_code": tbl.Code,
		"page_size":  1000,
	})
	paged = mustDecode[models.PagedRows](t, res)
	if paged.PageSize != 500 {
		t.Errorf("expected pageSize=500, got %d", paged.PageSize)
	}

	res = call(t, env.h, env.ws, "query_rows", map[string]any{
		"table_code": tbl.Code,
		"search":     "Alpha",
		"sorts":      []map[string]string{{"column": nameCol.Code, "direction": "asc"}},
		"filters":    []map[string]any{{"column": nameCol.Code, "operator": "is", "value": "Alpha"}},
	})
	paged = mustDecode[models.PagedRows](t, res)
	if paged.Total != 1 {
		t.Errorf("expected total=1 for search, got %d", paged.Total)
	}
}

func TestGetRowValidation(t *testing.T) {
	env := newEnv(t)
	res := call(t, env.h, env.ws, "get_row", map[string]any{"table_code": "NONEXISTENT", "row_id": 1})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "table not found") {
		t.Fatalf("expected 'table not found', got %v", res)
	}
}

func TestCreateRowValidation(t *testing.T) {
	env := newEnv(t)
	res := call(t, env.h, env.ws, "create_row", map[string]any{"table_code": "NONEXISTENT"})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "table not found") {
		t.Fatalf("expected 'table not found', got %v", res)
	}

	tbl := setupTableWithColumn(t, env)
	row := mustDecode[models.Row](t, call(t, env.h, env.ws, "create_row", map[string]any{
		"table_code": tbl.Code,
	}))
	if row.ID == 0 {
		t.Fatal("expected non-zero ID for empty data row")
	}
}

func TestPatchRowValidation(t *testing.T) {
	env := newEnv(t)
	res := call(t, env.h, env.ws, "patch_row", map[string]any{
		"table_code": "NONEXISTENT",
		"row_id":     1,
		"data":       json.RawMessage(`{}`),
	})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "table not found") {
		t.Fatalf("expected 'table not found', got %v", res)
	}
}

func TestBulkPatchRowsValidation(t *testing.T) {
	env := newEnv(t)
	res := call(t, env.h, env.ws, "bulk_patch_rows", map[string]any{
		"table_code": "NONEXISTENT",
		"filters":    []any{},
		"data":       json.RawMessage(`{}`),
	})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "table not found") {
		t.Fatalf("expected 'table not found', got %v", res)
	}
}

func TestDeleteRowValidationAndReferenceCheck(t *testing.T) {
	env := newEnv(t)
	tblA := setupTableWithColumn(t, env)
	tblB := mustDecode[models.Table](t, call(t, env.h, env.ws, "create_table", map[string]any{"name": "TableB"}))

	res := call(t, env.h, env.ws, "delete_row", map[string]any{"table_code": "NONEXISTENT", "row_id": 1})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "table not found") {
		t.Fatalf("expected 'table not found', got %v", res)
	}

	res = call(t, env.h, env.ws, "delete_row", map[string]any{"table_code": tblA.Code, "row_id": 99999})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "row not found: 99999") {
		t.Fatalf("expected 'row not found: 99999', got %v", res)
	}

	rowA := mustDecode[models.Row](t, call(t, env.h, env.ws, "create_row", map[string]any{"table_code": tblA.Code}))

	linkCol := mustDecode[models.Column](t, call(t, env.h, env.ws, "create_column", map[string]any{
		"table_code": tblB.Code,
		"name":       "link_to_a",
		"type":       "row-link",
		"options":    map[string]any{"targetTableCode": tblA.Code},
	}))

	linkData, _ := json.Marshal(map[string]any{
		linkCol.Code: map[string]any{"id": rowA.ID, "label": "Row A Label"},
	})
	rowB := mustDecode[models.Row](t, call(t, env.h, env.ws, "create_row", map[string]any{
		"table_code": tblB.Code,
		"data":       json.RawMessage(linkData),
	}))
	if rowB.ID == 0 {
		t.Fatal("expected rowB created")
	}

	res = call(t, env.h, env.ws, "delete_row", map[string]any{"table_code": tblA.Code, "row_id": rowA.ID})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "cannot be deleted") {
		t.Fatalf("expected reference error blocking delete, got %v", res)
	}

	refRes := call(t, env.h, env.ws, "list_referencing_rows", map[string]any{
		"table_code": tblA.Code,
		"row_id":     rowA.ID,
		"page":       0,
		"page_size":  0,
	})
	if refRes.IsError {
		t.Fatalf("list_referencing_rows failed: %s", refRes.Content[0].Text)
	}
	var pagedRefs struct {
		Total int `json:"total"`
		Data  []struct {
			SourceRowID     int64  `json:"source_row_id"`
			SourceTableCode string `json:"source_table_code"`
			ViaColumnCode   string `json:"via_column_code"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(refRes.Content[0].Text), &pagedRefs); err != nil {
		t.Fatalf("decode refs: %v", err)
	}
	if pagedRefs.Total != 1 {
		t.Errorf("expected 1 referencing row, got %d", pagedRefs.Total)
	}
	if len(pagedRefs.Data) > 0 {
		if pagedRefs.Data[0].SourceRowID != rowB.ID {
			t.Errorf("expected source_row_id %d, got %d", rowB.ID, pagedRefs.Data[0].SourceRowID)
		}
		if pagedRefs.Data[0].SourceTableCode != tblB.Code {
			t.Errorf("expected source_table_code %s, got %s", tblB.Code, pagedRefs.Data[0].SourceTableCode)
		}
		if pagedRefs.Data[0].ViaColumnCode != linkCol.Code {
			t.Errorf("expected via_column_code %s, got %s", linkCol.Code, pagedRefs.Data[0].ViaColumnCode)
		}
	}
}

func TestGetRowHistoryValidation(t *testing.T) {
	env := newEnv(t)
	res := call(t, env.h, env.ws, "get_row_history", map[string]any{
		"table_code": "NONEXISTENT",
		"row_id":     1,
	})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "table not found") {
		t.Fatalf("expected 'table not found', got %v", res)
	}
}

func TestAnnotateRowValidation(t *testing.T) {
	env := newEnv(t)
	tbl := setupTableWithColumn(t, env)
	row := mustDecode[models.Row](t, call(t, env.h, env.ws, "create_row", map[string]any{"table_code": tbl.Code}))

	res := call(t, env.h, env.ws, "annotate_row", map[string]any{
		"table_code": tbl.Code,
		"row_id":     row.ID,
		"text":       "",
	})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "text is required") {
		t.Fatalf("expected 'text is required', got %v", res)
	}

	res = call(t, env.h, env.ws, "annotate_row", map[string]any{
		"table_code": "NONEXISTENT",
		"row_id":     row.ID,
		"text":       "Hello",
	})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "table not found") {
		t.Fatalf("expected 'table not found', got %v", res)
	}
}

func TestCreateScriptValidation(t *testing.T) {
	env := newEnv(t)
	tbl := setupTableWithColumn(t, env)

	res := call(t, env.h, env.ws, "create_script", map[string]any{
		"name":       "",
		"event_type": "record.created",
		"code":       "print(1)",
	})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "required") {
		t.Fatalf("expected error for empty name, got %v", res)
	}

	res = call(t, env.h, env.ws, "create_script", map[string]any{
		"name":       "Script",
		"event_type": "",
		"code":       "print(1)",
	})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "required") {
		t.Fatalf("expected error for empty event_type, got %v", res)
	}

	res = call(t, env.h, env.ws, "create_script", map[string]any{
		"name":       "Script",
		"event_type": "record.created",
		"code":       "",
	})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "required") {
		t.Fatalf("expected error for empty code, got %v", res)
	}

	res = call(t, env.h, env.ws, "create_script", map[string]any{
		"name":        "Script",
		"event_type":  "record.created",
		"code":        "print(1)",
		"table_codes": []string{"NONEXISTENT"},
	})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "table not found: NONEXISTENT") {
		t.Fatalf("expected 'table not found: NONEXISTENT', got %v", res)
	}

	enabled := false
	s := mustDecode[models.Script](t, call(t, env.h, env.ws, "create_script", map[string]any{
		"name":        "Disabled Script",
		"event_type":  "record.created",
		"code":        "print(1)",
		"enabled":     &enabled,
		"table_codes": []string{tbl.Code},
	}))
	if s.Enabled {
		t.Error("expected enabled=false")
	}
	if len(s.TableIDs) != 1 || s.TableIDs[0] != tbl.ID {
		t.Errorf("expected tableIDs [%d], got %v", tbl.ID, s.TableIDs)
	}
}

func TestUpdateScriptValidation(t *testing.T) {
	env := newEnv(t)
	tbl := setupTableWithColumn(t, env)
	script := mustDecode[models.Script](t, call(t, env.h, env.ws, "create_script", map[string]any{
		"name":       "Script",
		"event_type": "record.created",
		"code":       "print(1)",
	}))

	res := call(t, env.h, env.ws, "update_script", map[string]any{
		"script_id": 99999,
	})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "script not found: 99999") {
		t.Fatalf("expected 'script not found: 99999', got %v", res)
	}

	res = call(t, env.h, env.ws, "update_script", map[string]any{
		"script_id":   script.ID,
		"table_codes": []string{"NONEXISTENT"},
	})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "table not found: NONEXISTENT") {
		t.Fatalf("expected 'table not found: NONEXISTENT', got %v", res)
	}

	newName := "Updated Name"
	newEvt := "record.updated"
	updated := mustDecode[models.Script](t, call(t, env.h, env.ws, "update_script", map[string]any{
		"script_id":   script.ID,
		"name":        &newName,
		"event_type":  &newEvt,
		"table_codes": []string{tbl.Code},
	}))
	if updated.Name != newName {
		t.Errorf("expected name %s, got %s", newName, updated.Name)
	}
	if string(updated.EventType) != newEvt {
		t.Errorf("expected eventType %s, got %s", newEvt, updated.EventType)
	}
	if len(updated.TableIDs) != 1 || updated.TableIDs[0] != tbl.ID {
		t.Errorf("expected tableIDs [%d], got %v", tbl.ID, updated.TableIDs)
	}
}

func TestDeleteScriptNotFound(t *testing.T) {
	env := newEnv(t)
	res := call(t, env.h, env.ws, "delete_script", map[string]any{"script_id": 99999})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "script not found: 99999") {
		t.Fatalf("expected 'script not found: 99999', got %v", res)
	}
}

func TestListScriptExecutionsValidation(t *testing.T) {
	env := newEnv(t)
	script := mustDecode[models.Script](t, call(t, env.h, env.ws, "create_script", map[string]any{
		"name":       "Script",
		"event_type": "record.created",
		"code":       "print(1)",
	}))

	res := call(t, env.h, env.ws, "list_script_executions", map[string]any{"script_id": 99999})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "script not found: 99999") {
		t.Fatalf("expected 'script not found: 99999', got %v", res)
	}

	ws2, err := env.ident.CreateWorkspace(context.Background(), "OtherWS", "", "OTHWS")
	if err != nil {
		t.Fatalf("create ws2: %v", err)
	}
	res = call(t, env.h, ws2, "list_script_executions", map[string]any{"script_id": script.ID})
	if !res.IsError || !strings.Contains(res.Content[0].Text, "script not found") {
		t.Fatalf("expected 'script not found' for ws2, got %v", res)
	}

	res = call(t, env.h, env.ws, "list_script_executions", map[string]any{"script_id": script.ID, "limit": 0})
	if res.IsError {
		t.Fatalf("list_script_executions failed with limit 0: %s", res.Content[0].Text)
	}
	res = call(t, env.h, env.ws, "list_script_executions", map[string]any{"script_id": script.ID, "limit": 500})
	if res.IsError {
		t.Fatalf("list_script_executions failed with limit 500: %s", res.Content[0].Text)
	}
}
