package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	lua "github.com/yuin/gopher-lua"

	"github.com/saintedlama/degubase/internal/infrastructure/events"
	"github.com/saintedlama/degubase/internal/models"
)

// execCounter gives each in-flight execution a stable non-zero context ID
// without a DB round-trip. The actual DB record ID is assigned at the end.
var execCounter atomic.Int64

type ScriptRunner struct {
	ctx        context.Context // cancelled on server shutdown
	auto       Store
	recs       RowMutator
	cols       ColumnLister
	broker     *events.Broker
	httpClient *http.Client
}

func NewScriptRunner(ctx context.Context, a Store, r RowMutator, c ColumnLister, b *events.Broker, policy HTTPPolicy) *ScriptRunner {
	return &ScriptRunner{ctx: ctx, auto: a, recs: r, cols: c, broker: b, httpClient: newSafeHTTPClient(policy)}
}

// Dispatch finds all enabled scripts matching table+event and fires each in
// its own goroutine. Returns immediately so the HTTP response is never blocked.
func (sr *ScriptRunner) Dispatch(ctx context.Context, ws *models.Workspace, table *models.Table, event models.ScriptEventType, row *models.Row) {
	scripts, err := sr.auto.ListScripts(ctx, ws.ID)
	if err != nil {
		slog.Error("script dispatch: list scripts", "workspace_id", ws.ID, "error", err)
		return
	}
	for _, sc := range scripts {
		if !sc.Enabled || sc.EventType != event || !scriptMatchesTable(sc, table.ID) {
			continue
		}
		s := sc
		go sr.run(sr.ctx, ws, table, &s, event, row)
	}
}

func scriptMatchesTable(s models.Script, tableID int64) bool {
	if len(s.TableIDs) == 0 {
		return true // workspace-wide script applies to every table
	}
	for _, id := range s.TableIDs {
		if id == tableID {
			return true
		}
	}
	return false
}

func (sr *ScriptRunner) run(ctx context.Context, ws *models.Workspace, table *models.Table, script *models.Script, event models.ScriptEventType, row *models.Row) {
	started := time.Now()
	var logs []models.ScriptLogEntry
	success := true

	ctxID := execCounter.Add(1)
	execCtx, cancel := context.WithTimeout(WithScriptExecution(ctx, ctxID), 30*time.Second)
	defer cancel()

	addLog := func(level models.ScriptLogLevel, msg string) {
		logs = append(logs, models.ScriptLogEntry{Level: level, Message: msg, Ts: time.Now()})
	}

	envVars, _ := sr.auto.ListScriptEnvVars(execCtx, ws.ID)

	// Build column name ↔ ID translation maps so scripts can use human-readable
	// column names while the store uses numeric IDs as JSON keys.
	cols, _ := sr.cols.ListColumns(execCtx, table.ID)
	nameToID := make(map[string]int64, len(cols))
	idToName := make(map[string]string, len(cols))
	for _, c := range cols {
		nameToID[c.Name] = c.ID
		idToName[strconv.FormatInt(c.ID, 10)] = c.Name
	}

	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer L.Close()

	lua.OpenBase(L)
	lua.OpenTable(L)
	lua.OpenString(L)
	lua.OpenMath(L)

	// rowDataToLua converts stored row JSON (ID-keyed) to a Lua table (name-keyed).
	rowDataToLua := func(data json.RawMessage) *lua.LTable {
		tbl := L.NewTable()
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			return tbl
		}
		for k, v := range m {
			if name, ok := idToName[k]; ok {
				k = name
			}
			L.SetField(tbl, k, anyToLua(L, v))
		}
		return tbl
	}

	// luaDataToJSON converts a name-keyed Lua table to ID-keyed JSON for storage.
	luaDataToJSON := func(tbl *lua.LTable) (json.RawMessage, error) {
		m := make(map[string]any)
		tbl.ForEach(func(k, v lua.LValue) {
			key := k.String()
			if id, ok := nameToID[key]; ok {
				key = strconv.FormatInt(id, 10)
			}
			m[key] = luaToAny(v)
		})
		return json.Marshal(m)
	}

	// print → execution log
	L.SetGlobal("print", L.NewFunction(func(L *lua.LState) int {
		parts := make([]string, L.GetTop())
		for i := range parts {
			parts[i] = L.Get(i + 1).String()
		}
		addLog(models.ScriptLogInfo, strings.Join(parts, "\t"))
		return 0
	}))

	// ── Event context ──────────────────────────────────────────────────────────
	// Flat globals for quick access
	L.SetGlobal("event_type", lua.LString(event))
	L.SetGlobal("table_name", lua.LString(table.Name))
	L.SetGlobal("table_code", lua.LString(table.Code))

	var triggerRowID int64
	eventDataTbl := L.NewTable()
	if row != nil {
		triggerRowID = row.ID
		L.SetGlobal("row_id", lua.LNumber(row.ID))
		if len(row.Data) > 0 {
			eventDataTbl = rowDataToLua(row.Data)
			L.SetGlobal("row", eventDataTbl) // backwards-compat alias
		}
	}

	// event table: event.type / event.data / event.row_id / event.table_name
	eventTbl := L.NewTable()
	L.SetField(eventTbl, "type", lua.LString(event))
	L.SetField(eventTbl, "table_name", lua.LString(table.Name))
	L.SetField(eventTbl, "table_code", lua.LString(table.Code))
	L.SetField(eventTbl, "row_id", lua.LNumber(triggerRowID))
	L.SetField(eventTbl, "data", eventDataTbl)
	L.SetGlobal("event", eventTbl)

	// ── Environment variables ──────────────────────────────────────────────────
	// Injected as globals (as per plan) AND accessible via env("KEY").
	for _, ev := range envVars {
		L.SetGlobal(ev.Key, lua.LString(ev.Value))
	}
	L.SetGlobal("env", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		for _, ev := range envVars {
			if ev.Key == key {
				L.Push(lua.LString(ev.Value))
				return 1
			}
		}
		L.Push(lua.LNil)
		return 1
	}))

	// ── log table ─────────────────────────────────────────────────────────────
	// log.info / log.warn / log.error  AND  log("msg") via __call metamethod.
	logTbl := L.NewTable()
	logMt := L.NewTable()
	L.SetField(logMt, "__call", L.NewFunction(func(L *lua.LState) int {
		L.Remove(1) // remove self
		if L.GetTop() >= 2 {
			addLog(models.ScriptLogLevel(L.CheckString(1)), L.CheckString(2))
		} else {
			addLog(models.ScriptLogInfo, L.CheckString(1))
		}
		return 0
	}))
	L.SetMetatable(logTbl, logMt)
	L.SetField(logTbl, "info", L.NewFunction(func(L *lua.LState) int {
		addLog(models.ScriptLogInfo, L.CheckString(1))
		return 0
	}))
	L.SetField(logTbl, "warn", L.NewFunction(func(L *lua.LState) int {
		addLog(models.ScriptLogWarn, L.CheckString(1))
		return 0
	}))
	L.SetField(logTbl, "error", L.NewFunction(func(L *lua.LState) int {
		addLog(models.ScriptLogError, L.CheckString(1))
		return 0
	}))
	L.SetGlobal("log", logTbl)

	// ── http table ────────────────────────────────────────────────────────────
	// http.get(url) / http.post(url, body) → {status, body} or nil, err
	httpTbl := L.NewTable()

	httpResponse := func(resp *http.Response, err error) int {
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		result := L.NewTable()
		L.SetField(result, "status", lua.LNumber(resp.StatusCode))
		L.SetField(result, "body", lua.LString(body))
		L.Push(result)
		return 1
	}

	doRequest := func(method, rawURL string, body io.Reader) (*http.Response, error) {
		if err := validateRequestURL(rawURL); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(execCtx, method, rawURL, body)
		if err != nil {
			return nil, err
		}
		if method == http.MethodPost {
			req.Header.Set("Content-Type", "application/json")
		}
		return sr.httpClient.Do(req)
	}

	L.SetField(httpTbl, "get", L.NewFunction(func(L *lua.LState) int {
		resp, err := doRequest(http.MethodGet, L.CheckString(1), nil)
		return httpResponse(resp, err)
	}))
	L.SetField(httpTbl, "post", L.NewFunction(func(L *lua.LState) int {
		resp, err := doRequest(http.MethodPost, L.CheckString(1), strings.NewReader(L.CheckString(2)))
		return httpResponse(resp, err)
	}))
	L.SetGlobal("http", httpTbl)

	// ── degubase table ────────────────────────────────────────────────────────
	// degubase.update_field(col, value)  — patch a single field on the triggering row
	// degubase.update_row(id, data)      — replace row data by ID
	// degubase.get_row(id)               — fetch row data by ID
	// degubase.create_row(data)          — create a new row in the current table
	// db.* kept as alias for backwards compat
	degubaseTbl := L.NewTable()

	luaGetRow := L.NewFunction(func(L *lua.LState) int {
		id := int64(L.CheckNumber(1))
		r, err := sr.recs.GetRow(execCtx, id)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(rowDataToLua(r.Data))
		return 1
	})

	luaUpdateRow := L.NewFunction(func(L *lua.LState) int {
		id := int64(L.CheckNumber(1))
		data, err := luaDataToJSON(L.CheckTable(2))
		if err != nil {
			addLog(models.ScriptLogError, "update_row: "+err.Error())
			L.Push(lua.LFalse)
			return 1
		}
		updated, err := sr.recs.UpdateRow(execCtx, id, data)
		if err != nil {
			addLog(models.ScriptLogError, "update_row: "+err.Error())
			L.Push(lua.LFalse)
			return 1
		}
		src, cmd := CommandSourceFromCtx(execCtx, "", "")
		sr.broker.Publish(table.ID, events.RowEvent{Type: events.RowEventUpdated, Row: updated, CommandSourceID: src, CommandID: cmd})
		L.Push(lua.LTrue)
		return 1
	})

	luaCreateRow := L.NewFunction(func(L *lua.LState) int {
		data, err := luaDataToJSON(L.CheckTable(1))
		if err != nil {
			addLog(models.ScriptLogError, "create_row: "+err.Error())
			L.Push(lua.LNil)
			return 1
		}
		created, err := sr.recs.CreateRow(execCtx, table.ID, data)
		if err != nil {
			addLog(models.ScriptLogError, "create_row: "+err.Error())
			L.Push(lua.LNil)
			return 1
		}
		src, cmd := CommandSourceFromCtx(execCtx, "", "")
		sr.broker.Publish(table.ID, events.RowEvent{Type: events.RowEventCreated, Row: created, CommandSourceID: src, CommandID: cmd})
		L.Push(lua.LNumber(created.ID))
		return 1
	})

	// update_field: patches a single named field on the triggering row
	L.SetField(degubaseTbl, "update_field", L.NewFunction(func(L *lua.LState) int {
		if triggerRowID == 0 {
			addLog(models.ScriptLogError, "degubase.update_field: no triggering row")
			L.Push(lua.LFalse)
			return 1
		}
		colName := L.CheckString(1)
		val := L.Get(2)

		// Resolve column name → storage key (numeric ID as string)
		dataKey := colName
		if id, ok := nameToID[colName]; ok {
			dataKey = strconv.FormatInt(id, 10)
		}

		current, err := sr.recs.GetRow(execCtx, triggerRowID)
		if err != nil {
			addLog(models.ScriptLogError, "degubase.update_field: "+err.Error())
			L.Push(lua.LFalse)
			return 1
		}
		var m map[string]any
		if err := json.Unmarshal(current.Data, &m); err != nil {
			m = make(map[string]any)
		}
		m[dataKey] = luaToAny(val)
		jsonData, err := json.Marshal(m)
		if err != nil {
			addLog(models.ScriptLogError, "degubase.update_field: "+err.Error())
			L.Push(lua.LFalse)
			return 1
		}
		updated, err := sr.recs.UpdateRow(execCtx, triggerRowID, jsonData)
		if err != nil {
			addLog(models.ScriptLogError, "degubase.update_field: "+err.Error())
			L.Push(lua.LFalse)
			return 1
		}
		src, cmd := CommandSourceFromCtx(execCtx, "", "")
		sr.broker.Publish(table.ID, events.RowEvent{Type: events.RowEventUpdated, Row: updated, CommandSourceID: src, CommandID: cmd})
		L.Push(lua.LTrue)
		return 1
	}))

	L.SetField(degubaseTbl, "get_row", luaGetRow)
	L.SetField(degubaseTbl, "update_row", luaUpdateRow)
	L.SetField(degubaseTbl, "create_row", luaCreateRow)
	L.SetGlobal("degubase", degubaseTbl)

	// db kept as alias
	dbTbl := L.NewTable()
	L.SetField(dbTbl, "get_row", luaGetRow)
	L.SetField(dbTbl, "update_row", luaUpdateRow)
	L.SetField(dbTbl, "create_row", luaCreateRow)
	L.SetGlobal("db", dbTbl)

	if err := L.DoString(script.Code); err != nil {
		success = false
		addLog(models.ScriptLogError, err.Error())
	}

	ended := time.Now()
	if logs == nil {
		logs = []models.ScriptLogEntry{}
	}

	var rid *int64
	if triggerRowID != 0 {
		rid = &triggerRowID
	}
	if _, err := sr.auto.CreateScriptExecution(sr.ctx, models.ScriptExecution{
		ScriptID:   script.ID,
		RowID:      rid,
		TableCode:  table.Code,
		EventType:  event,
		StartedAt:  started,
		EndedAt:    ended,
		DurationMs: ended.Sub(started).Milliseconds(),
		Success:    success,
		Logs:       logs,
	}); err != nil {
		slog.Error("script execution: save record", "script_id", script.ID, "error", err)
	}
}

// ── JSON ↔ Lua helpers ────────────────────────────────────────────────────────

func anyToLua(L *lua.LState, v any) lua.LValue {
	switch val := v.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(val)
	case float64:
		return lua.LNumber(val)
	case string:
		return lua.LString(val)
	case map[string]any:
		tbl := L.NewTable()
		for k, mv := range val {
			L.SetField(tbl, k, anyToLua(L, mv))
		}
		return tbl
	case []any:
		tbl := L.NewTable()
		for i, av := range val {
			tbl.RawSetInt(i+1, anyToLua(L, av))
		}
		return tbl
	default:
		return lua.LString(fmt.Sprintf("%v", val))
	}
}

func luaTableToMap(tbl *lua.LTable) map[string]any {
	m := make(map[string]any)
	tbl.ForEach(func(key, val lua.LValue) {
		m[key.String()] = luaToAny(val)
	})
	return m
}

func luaToAny(v lua.LValue) any {
	switch val := v.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(val)
	case lua.LNumber:
		return float64(val)
	case lua.LString:
		return string(val)
	case *lua.LTable:
		return luaTableToMap(val)
	default:
		return val.String()
	}
}
