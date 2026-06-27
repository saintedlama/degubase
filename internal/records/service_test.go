package records_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/saintedlama/degubase/internal/testutil"
)

// TestRowServiceTranslation verifies that the RowService correctly translates
// code-keyed request data to ID-keyed storage data and back via the HTTP API.
func TestRowServiceTranslation(t *testing.T) {
	srv := testutil.NewServer(t)

	// Create workspace + table + column via API (integration path uses RowService)
	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "svc-test"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "things"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")

	cb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode), map[string]any{
		"name": "title",
		"type": "text",
	}, http.StatusCreated)
	columnCode := testutil.Field(t, testutil.Obj(t, cb), "code")
	if columnCode != "title" {
		t.Fatalf("expected code 'title', got %q", columnCode)
	}

	rowsBase := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode)

	t.Run("create with code key", func(t *testing.T) {
		data, _ := json.Marshal(map[string]any{"title": "hello"})
		body := map[string]json.RawMessage{"data": data}
		rb := testutil.MustReq(t, srv, "POST", rowsBase, body, http.StatusCreated)
		o := testutil.Obj(t, rb)
		rowData := testutil.Obj(t, o["data"])
		if rowData["title"] == nil {
			t.Errorf("expected 'title' key in response data, got %v", rowData)
		}
	})

	t.Run("list returns code keys", func(t *testing.T) {
		rb := testutil.MustReq(t, srv, "GET", rowsBase, nil, http.StatusOK)
		o := testutil.Obj(t, rb)
		rows := testutil.Arr(t, o["data"])
		if len(rows) == 0 {
			t.Fatal("expected at least 1 row")
		}
		firstRow := testutil.Obj(t, rows[0])
		rowData := testutil.Obj(t, firstRow["data"])
		if rowData["title"] == nil {
			t.Errorf("expected 'title' key in listed row data, got %v", rowData)
		}
	})
}
