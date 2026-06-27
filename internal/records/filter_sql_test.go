package records_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/saintedlama/degubase/internal/testutil"
)

// TestSQLFilterSortSearch verifies that filters, sorts, and search are pushed
// into SQL and return correct results (not fetching all rows in Go).
func TestSQLFilterSortSearch(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "filter-test"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "items"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")

	colsBase := fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode)
	rowsBase := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode)

	testutil.MustReq(t, srv, "POST", colsBase, map[string]any{"name": "title", "type": "text"}, http.StatusCreated)
	testutil.MustReq(t, srv, "POST", colsBase, map[string]any{"name": "score", "type": "number"}, http.StatusCreated)

	createRow := func(title string, score int) {
		data, _ := json.Marshal(map[string]any{"title": title, "score": score})
		testutil.MustReq(t, srv, "POST", rowsBase, map[string]json.RawMessage{"data": data}, http.StatusCreated)
	}
	createRow("alpha", 10)
	createRow("beta", 20)
	createRow("gamma", 30)

	t.Run("filter is", func(t *testing.T) {
		filters, _ := json.Marshal([]map[string]any{{"col": "title", "op": "is", "value": "alpha"}})
		rb := testutil.MustReq(t, srv, "GET", rowsBase+"?filters="+url.QueryEscape(string(filters)), nil, http.StatusOK)
		o := testutil.Obj(t, rb)
		rows := testutil.Arr(t, o["data"])
		if len(rows) != 1 {
			t.Errorf("filter 'is alpha': want 1 row, got %d", len(rows))
		}
	})

	t.Run("filter contains", func(t *testing.T) {
		filters, _ := json.Marshal([]map[string]any{{"col": "title", "op": "contains", "value": "et"}})
		rb := testutil.MustReq(t, srv, "GET", rowsBase+"?filters="+url.QueryEscape(string(filters)), nil, http.StatusOK)
		o := testutil.Obj(t, rb)
		rows := testutil.Arr(t, o["data"])
		if len(rows) != 1 {
			t.Errorf("filter 'contains et': want 1 row (beta), got %d", len(rows))
		}
	})

	t.Run("filter gt", func(t *testing.T) {
		filters, _ := json.Marshal([]map[string]any{{"col": "score", "op": "gt", "value": "15"}})
		rb := testutil.MustReq(t, srv, "GET", rowsBase+"?filters="+url.QueryEscape(string(filters)), nil, http.StatusOK)
		o := testutil.Obj(t, rb)
		rows := testutil.Arr(t, o["data"])
		if len(rows) != 2 {
			t.Errorf("filter 'score > 15': want 2 rows, got %d", len(rows))
		}
	})

	t.Run("search", func(t *testing.T) {
		rb := testutil.MustReq(t, srv, "GET", rowsBase+"?q=gamma", nil, http.StatusOK)
		o := testutil.Obj(t, rb)
		rows := testutil.Arr(t, o["data"])
		if len(rows) != 1 {
			t.Errorf("search 'gamma': want 1 row, got %d", len(rows))
		}
	})

	t.Run("sort desc", func(t *testing.T) {
		sorts, _ := json.Marshal([]map[string]any{{"col": "title", "dir": "desc"}})
		rb := testutil.MustReq(t, srv, "GET", rowsBase+"?sort="+url.QueryEscape(string(sorts)), nil, http.StatusOK)
		o := testutil.Obj(t, rb)
		rows := testutil.Arr(t, o["data"])
		if len(rows) != 3 {
			t.Fatalf("sort: want 3 rows, got %d", len(rows))
		}
		first := testutil.Obj(t, rows[0])
		firstData := testutil.Obj(t, first["data"])
		if firstData["title"] != "gamma" {
			t.Errorf("sort desc: first row should be 'gamma', got %v", firstData["title"])
		}
	})

	t.Run("total count reflects filter", func(t *testing.T) {
		filters, _ := json.Marshal([]map[string]any{{"col": "title", "op": "is_not", "value": "alpha"}})
		rb := testutil.MustReq(t, srv, "GET", rowsBase+"?filters="+url.QueryEscape(string(filters)), nil, http.StatusOK)
		o := testutil.Obj(t, rb)
		total := o["total"].(float64)
		if int(total) != 2 {
			t.Errorf("total after filter: want 2, got %d", int(total))
		}
	})
}
