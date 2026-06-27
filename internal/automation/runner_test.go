package automation_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/saintedlama/degubase/internal/testutil"
)

// pollField retries GET rowPath every 50 ms until data[field] == want, or 3 s elapses.
func pollField(t *testing.T, srv *httptest.Server, rowPath, field, want string) bool {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		code, body := testutil.Req(t, srv, "GET", rowPath, nil)
		if code == http.StatusOK {
			data := testutil.Obj(t, testutil.Obj(t, body)["data"])
			if v, _ := data[field].(string); v == want {
				return true
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// TestScriptRunnerDispatch verifies that enabled Lua scripts fire on row events.
func TestScriptRunnerDispatch(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "auto"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "things"}, http.StatusCreated)
	tblID := testutil.Obj(t, tb)["id"].(float64)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")

	colsBase := fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode)
	rowsBase := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode)
	scriptsBase := fmt.Sprintf("/api/workspaces/%s/scripts/", wsCode)

	testutil.MustReq(t, srv, "POST", colsBase, map[string]any{"name": "input", "type": "text"}, http.StatusCreated)
	testutil.MustReq(t, srv, "POST", colsBase, map[string]any{"name": "output", "type": "text"}, http.StatusCreated)

	// record.created: copy input → output
	testutil.MustReq(t, srv, "POST", scriptsBase, map[string]any{
		"name":       "copy on create",
		"event_type": "record.created",
		"enabled":    true,
		"table_ids":  []float64{tblID},
		"code":       `degubase.update_field("output", event.data.input or "")`,
	}, http.StatusCreated)

	// record.updated: copy input → output with "-updated" suffix
	testutil.MustReq(t, srv, "POST", scriptsBase, map[string]any{
		"name":       "copy on update",
		"event_type": "record.updated",
		"enabled":    true,
		"table_ids":  []float64{tblID},
		"code":       `degubase.update_field("output", (event.data.input or "") .. "-updated")`,
	}, http.StatusCreated)

	t.Run("record.created triggers script and sets output", func(t *testing.T) {
		rb := testutil.MustReq(t, srv, "POST", rowsBase, map[string]any{
			"data": map[string]any{"input": "hello"},
		}, http.StatusCreated)
		rowID := testutil.Obj(t, rb)["id"].(float64)
		rowPath := fmt.Sprintf("%s%.0f", rowsBase, rowID)

		if !pollField(t, srv, rowPath, "output", "hello") {
			row := testutil.MustReq(t, srv, "GET", rowPath, nil, http.StatusOK)
			t.Errorf("record.created script: output = %v, want %q",
				testutil.Obj(t, testutil.Obj(t, row)["data"])["output"], "hello")
		}
	})

	t.Run("record.updated triggers script and sets output", func(t *testing.T) {
		rb := testutil.MustReq(t, srv, "POST", rowsBase, map[string]any{
			"data": map[string]any{"input": "world"},
		}, http.StatusCreated)
		rowID := testutil.Obj(t, rb)["id"].(float64)
		rowPath := fmt.Sprintf("%s%.0f", rowsBase, rowID)

		// Wait for the create script to finish before triggering update.
		pollField(t, srv, rowPath, "output", "world")

		testutil.MustReq(t, srv, "PATCH", rowPath, map[string]any{
			"data": map[string]any{"input": "world"},
		}, http.StatusOK)

		if !pollField(t, srv, rowPath, "output", "world-updated") {
			row := testutil.MustReq(t, srv, "GET", rowPath, nil, http.StatusOK)
			t.Errorf("record.updated script: output = %v, want %q",
				testutil.Obj(t, testutil.Obj(t, row)["data"])["output"], "world-updated")
		}
	})
}
