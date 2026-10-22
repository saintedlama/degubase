package automation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"

	"github.com/saintedlama/degubase/internal/infrastructure/events"
	"github.com/saintedlama/degubase/internal/models"
)

type mockAutoStore struct {
	Store
	scripts          []models.Script
	scriptsErr       error
	envVars          []models.ScriptEnvVar
	createdExecution *models.ScriptExecution
}

func (m *mockAutoStore) ListScripts(ctx context.Context, workspaceID int64) ([]models.Script, error) {
	if m.scriptsErr != nil {
		return nil, m.scriptsErr
	}
	return m.scripts, nil
}

func (m *mockAutoStore) ListScriptEnvVars(ctx context.Context, workspaceID int64) ([]models.ScriptEnvVar, error) {
	return m.envVars, nil
}

func (m *mockAutoStore) CreateScriptExecution(ctx context.Context, exec models.ScriptExecution) (*models.ScriptExecution, error) {
	m.createdExecution = &exec
	return &exec, nil
}

type mockRowMutator struct {
	RowMutator
	rows      map[int64]*models.Row
	getRowErr error
	createErr error
	updateErr error
	created   []json.RawMessage
	updated   map[int64]json.RawMessage
}

func newMockRowMutator() *mockRowMutator {
	return &mockRowMutator{
		rows:    make(map[int64]*models.Row),
		updated: make(map[int64]json.RawMessage),
	}
}

func (m *mockRowMutator) GetRow(ctx context.Context, id int64) (*models.Row, error) {
	if m.getRowErr != nil {
		return nil, m.getRowErr
	}
	r, ok := m.rows[id]
	if !ok {
		return nil, fmt.Errorf("row not found: %d", id)
	}
	return r, nil
}

func (m *mockRowMutator) CreateRow(ctx context.Context, tableID int64, data json.RawMessage) (*models.Row, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.created = append(m.created, data)
	row := &models.Row{ID: int64(len(m.created) + 100), TableID: tableID, Data: data}
	m.rows[row.ID] = row
	return row, nil
}

func (m *mockRowMutator) UpdateRow(ctx context.Context, id int64, data json.RawMessage) (*models.Row, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	m.updated[id] = data
	row := &models.Row{ID: id, Data: data}
	m.rows[id] = row
	return row, nil
}

type mockColLister struct {
	ColumnLister
	cols []models.Column
}

func (m *mockColLister) ListColumns(ctx context.Context, tableID int64) ([]models.Column, error) {
	return m.cols, nil
}

func TestAnyToLuaAndLuaToAny(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	// nil
	lvNil := anyToLua(L, nil)
	if lvNil != lua.LNil {
		t.Errorf("expected LNil, got %v", lvNil)
	}
	if v := luaToAny(lvNil); v != nil {
		t.Errorf("expected nil, got %v", v)
	}

	// bool
	lvBool := anyToLua(L, true)
	if lvBool != lua.LTrue {
		t.Errorf("expected LTrue, got %v", lvBool)
	}
	if v := luaToAny(lvBool); v != true {
		t.Errorf("expected true, got %v", v)
	}

	// float64
	lvNum := anyToLua(L, float64(42.5))
	if lvNum != lua.LNumber(42.5) {
		t.Errorf("expected LNumber(42.5), got %v", lvNum)
	}
	if v := luaToAny(lvNum); v != float64(42.5) {
		t.Errorf("expected 42.5, got %v", v)
	}

	// string
	lvStr := anyToLua(L, "hello")
	if lvStr != lua.LString("hello") {
		t.Errorf("expected LString, got %v", lvStr)
	}
	if v := luaToAny(lvStr); v != "hello" {
		t.Errorf("expected 'hello', got %v", v)
	}

	// map[string]any
	m := map[string]any{
		"nested_str": "val",
		"nested_num": float64(10),
	}
	lvTable := anyToLua(L, m)
	if lvTable.Type() != lua.LTTable {
		t.Fatalf("expected table, got %v", lvTable.Type())
	}
	vMap := luaToAny(lvTable)
	mOut, ok := vMap.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", vMap)
	}
	if mOut["nested_str"] != "val" || mOut["nested_num"] != float64(10) {
		t.Errorf("unexpected map contents: %v", mOut)
	}

	// []any
	arr := []any{"elem1", float64(2), true}
	lvArr := anyToLua(L, arr)
	if lvArr.Type() != lua.LTTable {
		t.Fatalf("expected table for array, got %v", lvArr.Type())
	}

	// default in anyToLua (e.g. int)
	lvInt := anyToLua(L, 12345)
	if lvInt != lua.LString("12345") {
		t.Errorf("expected LString '12345', got %v", lvInt)
	}

	// default in luaToAny (e.g. function)
	fn := L.NewFunction(func(l *lua.LState) int { return 0 })
	vFn := luaToAny(fn)
	if _, ok := vFn.(string); !ok {
		t.Errorf("expected string for fn, got %T", vFn)
	}
}

func TestScriptMatchesTable(t *testing.T) {
	sAll := models.Script{TableIDs: nil}
	if !scriptMatchesTable(sAll, 123) {
		t.Error("expected empty TableIDs to match table 123")
	}

	sSpecific := models.Script{TableIDs: []int64{10, 20}}
	if !scriptMatchesTable(sSpecific, 10) {
		t.Error("expected match for table 10")
	}
	if !scriptMatchesTable(sSpecific, 20) {
		t.Error("expected match for table 20")
	}
	if scriptMatchesTable(sSpecific, 30) {
		t.Error("did not expect match for table 30")
	}
}

func TestDispatchErrorAndFilters(t *testing.T) {
	ctx := context.Background()
	ws := &models.Workspace{ID: 1}
	tbl := &models.Table{ID: 10, Code: "TBL"}

	// ListScripts error returns early without panic
	autoErr := &mockAutoStore{scriptsErr: errors.New("db error")}
	runnerErr := NewScriptRunner(ctx, autoErr, newMockRowMutator(), &mockColLister{}, events.NewBroker(), HTTPPolicy{})
	runnerErr.Dispatch(ctx, ws, tbl, models.ScriptEventRecordCreated, nil)

	// List of scripts with various skip conditions: disabled, wrong event, wrong table
	autoFiltered := &mockAutoStore{
		scripts: []models.Script{
			{ID: 1, Enabled: false, EventType: models.ScriptEventRecordCreated, TableIDs: []int64{10}},
			{ID: 2, Enabled: true, EventType: models.ScriptEventRecordUpdated, TableIDs: []int64{10}},
			{ID: 3, Enabled: true, EventType: models.ScriptEventRecordCreated, TableIDs: []int64{99}},
		},
	}
	runnerFiltered := NewScriptRunner(ctx, autoFiltered, newMockRowMutator(), &mockColLister{}, events.NewBroker(), HTTPPolicy{})
	runnerFiltered.Dispatch(ctx, ws, tbl, models.ScriptEventRecordCreated, nil)
}

func TestScriptRunnerExecutionFeatures(t *testing.T) {
	ctx := context.Background()
	ws := &models.Workspace{ID: 1}
	tbl := &models.Table{ID: 10, Name: "Tasks", Code: "TASK"}
	cols := []models.Column{
		{ID: 101, Name: "title", Type: "text"},
		{ID: 102, Name: "status", Type: "text"},
	}

	// HTTP test server for http.get and http.post
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			body, _ := io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("created:" + string(body)))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok-get"))
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	policy := HTTPPolicy{AllowedHosts: []string{u.Host, "127.0.0.1"}}

	t.Run("globals, logs, and print", func(t *testing.T) {
		auto := &mockAutoStore{
			envVars: []models.ScriptEnvVar{
				{Key: "API_KEY", Value: "secret-123"},
			},
		}
		recs := newMockRowMutator()
		runner := NewScriptRunner(ctx, auto, recs, &mockColLister{cols: cols}, events.NewBroker(), policy)

		script := &models.Script{
			ID: 1,
			Code: `
print("arg1", "arg2")
log.info("info text")
log.warn("warn text")
log.error("error text")
log("called directly")
log("warn", "called with level")

assert(event_type == "record.created", "event_type mismatch")
assert(table_name == "Tasks", "table_name mismatch")
assert(table_code == "TASK", "table_code mismatch")
assert(row_id == 50, "row_id mismatch")
assert(row.title == "Initial Title", "row.title mismatch")
assert(event.row_id == 50, "event.row_id mismatch")
assert(event.data.title == "Initial Title", "event.data.title mismatch")

assert(API_KEY == "secret-123", "global env var mismatch")
assert(env("API_KEY") == "secret-123", "env() mismatch")
assert(env("NONEXISTENT") == nil, "missing env var should be nil")
`,
		}

		rowData, _ := json.Marshal(map[string]any{"101": "Initial Title"})
		row := &models.Row{ID: 50, TableID: tbl.ID, Data: rowData}

		runner.run(ctx, ws, tbl, script, models.ScriptEventRecordCreated, row)

		exec := auto.createdExecution
		if exec == nil {
			t.Fatal("expected script execution created")
		}
		if !exec.Success {
			t.Fatalf("expected execution success, logs: %v", exec.Logs)
		}

		foundPrint := false
		for _, l := range exec.Logs {
			if strings.Contains(l.Message, "arg1\targ2") {
				foundPrint = true
			}
		}
		if !foundPrint {
			t.Error("print() output not found in logs")
		}
	})

	t.Run("degubase.update_field success and errors", func(t *testing.T) {
		auto := &mockAutoStore{}
		recs := newMockRowMutator()
		recs.rows[50] = &models.Row{
			ID:      50,
			TableID: tbl.ID,
			Data:    json.RawMessage(`{"101":"Old Title"}`),
		}
		runner := NewScriptRunner(ctx, auto, recs, &mockColLister{cols: cols}, events.NewBroker(), policy)

		// 1. Without triggering row (triggerRowID == 0) -> update_field returns false
		scriptNoRow := &models.Script{
			ID: 2,
			Code: `
res = degubase.update_field("title", "New")
assert(res == false, "update_field without row should return false")
`,
		}
		runner.run(ctx, ws, tbl, scriptNoRow, models.ScriptEventRecordCreated, nil)
		if !auto.createdExecution.Success {
			t.Errorf("script failed: %v", auto.createdExecution.Logs)
		}

		// 2. Success with triggering row -> updates row data in store
		scriptWithRow := &models.Script{
			ID: 3,
			Code: `
res = degubase.update_field("title", "Updated Title")
assert(res == true, "update_field should return true")
`,
		}
		row := &models.Row{ID: 50, TableID: tbl.ID, Data: json.RawMessage(`{"101":"Old Title"}`)}
		runner.run(ctx, ws, tbl, scriptWithRow, models.ScriptEventRecordUpdated, row)
		if !auto.createdExecution.Success {
			t.Errorf("script failed: %v", auto.createdExecution.Logs)
		}
		updatedData := recs.updated[50]
		var m map[string]any
		json.Unmarshal(updatedData, &m)
		if m["101"] != "Updated Title" {
			t.Errorf("expected 101='Updated Title', got %v", m)
		}

		// 3. RowMutator error on update
		recs.updateErr = errors.New("db write failed")
		scriptErr := &models.Script{
			ID: 4,
			Code: `
res = degubase.update_field("title", "Boom")
assert(res == false, "update_field on error should return false")
`,
		}
		runner.run(ctx, ws, tbl, scriptErr, models.ScriptEventRecordUpdated, row)
	})

	t.Run("degubase and db methods: get_row, update_row, create_row", func(t *testing.T) {
		auto := &mockAutoStore{}
		recs := newMockRowMutator()
		recs.rows[10] = &models.Row{
			ID:      10,
			TableID: tbl.ID,
			Data:    json.RawMessage(`{"101":"Item 10","102":"Pending"}`),
		}
		runner := NewScriptRunner(ctx, auto, recs, &mockColLister{cols: cols}, events.NewBroker(), policy)

		script := &models.Script{
			ID: 5,
			Code: `
-- get_row
r = degubase.get_row(10)
assert(r.title == "Item 10", "get_row title mismatch")

-- get_row not found
nilRow, err = degubase.get_row(9999)
assert(nilRow == nil, "expected nil row")
assert(err ~= nil, "expected error for missing row")

-- update_row via degubase
upRes = degubase.update_row(10, {title = "Item 10 Updated", status = "Done"})
assert(upRes == true, "update_row should succeed")

-- create_row via degubase
newID = degubase.create_row({title = "New Item", status = "Active"})
assert(newID ~= nil, "create_row should return new ID")

-- test db.* aliases
r2 = db.get_row(10)
assert(r2 ~= nil, "db.get_row failed")
upRes2 = db.update_row(10, {title = "Item 10 via db"})
assert(upRes2 == true, "db.update_row failed")
newID2 = db.create_row({title = "New Item via db"})
assert(newID2 ~= nil, "db.create_row failed")
`,
		}
		runner.run(ctx, ws, tbl, script, models.ScriptEventRecordCreated, nil)
		if !auto.createdExecution.Success {
			t.Fatalf("script failed: %v", auto.createdExecution.Logs)
		}

		// Mutator errors
		recs.getRowErr = errors.New("get error")
		recs.updateErr = errors.New("update error")
		recs.createErr = errors.New("create error")
		scriptErrors := &models.Script{
			ID: 6,
			Code: `
r, err = degubase.get_row(10)
assert(r == nil and err ~= nil, "expected get_row error")

upRes = degubase.update_row(10, {title = "Fail"})
assert(upRes == false, "expected update_row false on error")

newID = degubase.create_row({title = "Fail"})
assert(newID == nil, "expected create_row nil on error")
`,
		}
		runner.run(ctx, ws, tbl, scriptErrors, models.ScriptEventRecordCreated, nil)
	})

	t.Run("http.get and http.post", func(t *testing.T) {
		auto := &mockAutoStore{}
		recs := newMockRowMutator()
		runner := NewScriptRunner(ctx, auto, recs, &mockColLister{cols: cols}, events.NewBroker(), policy)

		script := &models.Script{
			ID: 7,
			Code: fmt.Sprintf(`
-- Successful GET
res, err = http.get("%s/test-get")
assert(res ~= nil, "http.get failed: " .. tostring(err))
assert(res.status == 200, "expected status 200")
assert(res.body == "ok-get", "expected body ok-get")

-- Successful POST
res2, err2 = http.post("%s/test-post", "post-data")
assert(res2 ~= nil, "http.post failed: " .. tostring(err2))
assert(res2.status == 201, "expected status 201")
assert(res2.body == "created:post-data", "expected body created:post-data")

-- Invalid URL scheme
resErr, err3 = http.get("ftp://example.com")
assert(resErr == nil, "expected nil response for bad scheme")
assert(err3 ~= nil, "expected error for bad scheme")

-- Blocked SSRF URL (unallowed private IP)
resBlocked, errBlocked = http.get("http://192.168.1.1/secret")
assert(resBlocked == nil, "expected blocked SSRF")
assert(errBlocked ~= nil, "expected error on blocked SSRF")
`, srv.URL, srv.URL),
		}
		runner.run(ctx, ws, tbl, script, models.ScriptEventRecordCreated, nil)
		if !auto.createdExecution.Success {
			t.Fatalf("http script failed: %v", auto.createdExecution.Logs)
		}
	})

	t.Run("lua syntax and runtime errors", func(t *testing.T) {
		auto := &mockAutoStore{}
		recs := newMockRowMutator()
		runner := NewScriptRunner(ctx, auto, recs, &mockColLister{cols: cols}, events.NewBroker(), policy)

		// Syntax error
		scriptSyntax := &models.Script{
			ID:   8,
			Code: `function unclosed(`,
		}
		runner.run(ctx, ws, tbl, scriptSyntax, models.ScriptEventRecordCreated, nil)
		if auto.createdExecution.Success {
			t.Error("expected Success == false for syntax error")
		}

		// Runtime error
		scriptRuntime := &models.Script{
			ID:   9,
			Code: `error("explicit runtime failure")`,
		}
		runner.run(ctx, ws, tbl, scriptRuntime, models.ScriptEventRecordCreated, nil)
		if auto.createdExecution.Success {
			t.Error("expected Success == false for runtime error")
		}
	})
}
