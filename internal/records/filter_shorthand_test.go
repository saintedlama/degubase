package records_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/saintedlama/degubase/internal/testutil"
)

// TestShorthandFilters verifies the filter=col:op:value query-param format
// and the sort=col:dir shorthand, both of which bypass the JSON filter path.
func TestShorthandFilters(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "sf"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")
	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "items"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")
	colsBase := fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode)
	rowsBase := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode)

	testutil.MustReq(t, srv, "POST", colsBase, map[string]any{"name": "title", "type": "text"}, http.StatusCreated)
	testutil.MustReq(t, srv, "POST", colsBase, map[string]any{"name": "score", "type": "number"}, http.StatusCreated)
	testutil.MustReq(t, srv, "POST", colsBase, map[string]any{
		"name":    "status",
		"type":    "single-select",
		"options": map[string]any{"choices": []string{"open", "closed", "pending"}},
	}, http.StatusCreated)

	createRow := func(title string, score int, status string) {
		data, _ := json.Marshal(map[string]any{"title": title, "score": score, "status": status})
		testutil.MustReq(t, srv, "POST", rowsBase, map[string]json.RawMessage{"data": data}, http.StatusCreated)
	}
	createRow("alpha", 10, "open")
	createRow("beta", 20, "closed")
	createRow("gamma", 30, "open")
	createRow("delta", 40, "pending")

	rowCount := func(t *testing.T, query string) int {
		t.Helper()
		rb := testutil.MustReq(t, srv, "GET", rowsBase+"?"+query, nil, http.StatusOK)
		return len(testutil.Arr(t, testutil.Obj(t, rb)["data"]))
	}

	firstTitle := func(t *testing.T, query string) string {
		t.Helper()
		rb := testutil.MustReq(t, srv, "GET", rowsBase+"?"+query, nil, http.StatusOK)
		rows := testutil.Arr(t, testutil.Obj(t, rb)["data"])
		if len(rows) == 0 {
			t.Fatal("no rows returned")
		}
		return testutil.Obj(t, testutil.Obj(t, rows[0])["data"])["title"].(string)
	}

	// ── Shorthand two-part (implies "is") ──────────────────────────────────────

	t.Run("shorthand two-part implies is", func(t *testing.T) {
		if n := rowCount(t, "filter=status:open"); n != 2 {
			t.Errorf("filter=status:open: want 2, got %d", n)
		}
	})

	// ── Shorthand three-part explicit operator ──────────────────────────────────

	t.Run("shorthand is", func(t *testing.T) {
		if n := rowCount(t, "filter=title:is:alpha"); n != 1 {
			t.Errorf("filter=title:is:alpha: want 1, got %d", n)
		}
	})

	t.Run("shorthand is_not", func(t *testing.T) {
		if n := rowCount(t, "filter=status:is_not:open"); n != 2 {
			t.Errorf("filter=status:is_not:open: want 2, got %d", n)
		}
	})

	t.Run("shorthand contains", func(t *testing.T) {
		if n := rowCount(t, "filter=title:contains:et"); n != 1 {
			t.Errorf("filter=title:contains:et: want 1 (beta), got %d", n)
		}
	})

	t.Run("shorthand not_contains", func(t *testing.T) {
		// "lpha" appears only in "alpha"; not_contains should return the other 3
		if n := rowCount(t, "filter=title:not_contains:lpha"); n != 3 {
			t.Errorf("filter=title:not_contains:lpha: want 3, got %d", n)
		}
	})

	t.Run("shorthand gt", func(t *testing.T) {
		if n := rowCount(t, "filter=score:gt:20"); n != 2 {
			t.Errorf("filter=score:gt:20: want 2, got %d", n)
		}
	})

	t.Run("shorthand lt", func(t *testing.T) {
		if n := rowCount(t, "filter=score:lt:30"); n != 2 {
			t.Errorf("filter=score:lt:30: want 2 (alpha, beta), got %d", n)
		}
	})

	t.Run("shorthand eq", func(t *testing.T) {
		if n := rowCount(t, "filter=score:eq:20"); n != 1 {
			t.Errorf("filter=score:eq:20: want 1 (beta), got %d", n)
		}
	})

	t.Run("shorthand in with pipe-separated values", func(t *testing.T) {
		if n := rowCount(t, "filter=status:in:open|pending"); n != 3 {
			t.Errorf("filter=status:in:open|pending: want 3, got %d", n)
		}
	})

	t.Run("shorthand not_in", func(t *testing.T) {
		if n := rowCount(t, "filter=status:not_in:open|pending"); n != 1 {
			t.Errorf("filter=status:not_in:open|pending: want 1 (closed), got %d", n)
		}
	})

	// ── AND: multiple filter= params ────────────────────────────────────────────

	t.Run("multiple filters combined as AND", func(t *testing.T) {
		if n := rowCount(t, "filter=status:open&filter=score:gt:15"); n != 1 {
			t.Errorf("status=open AND score>15: want 1 (gamma), got %d", n)
		}
	})

	// ── Shorthand sort ──────────────────────────────────────────────────────────

	t.Run("shorthand sort asc default", func(t *testing.T) {
		title := firstTitle(t, "sort=title")
		if title != "alpha" {
			t.Errorf("sort=title asc: first want alpha, got %s", title)
		}
	})

	t.Run("shorthand sort desc", func(t *testing.T) {
		title := firstTitle(t, "sort=title:desc")
		if title != "gamma" {
			t.Errorf("sort=title:desc: first want gamma, got %s", title)
		}
	})

	t.Run("shorthand sort multi-column", func(t *testing.T) {
		// status asc (closed < open < pending), then score desc within group
		title := firstTitle(t, "sort=status:asc,score:desc")
		if title != "beta" {
			t.Errorf("sort=status:asc,score:desc: first want beta (closed), got %s", title)
		}
	})

	t.Run("shorthand sort invalid direction returns 400", func(t *testing.T) {
		code, _ := testutil.Req(t, srv, "GET", rowsBase+"?sort=title:sideways", nil)
		if code != http.StatusBadRequest {
			t.Errorf("invalid sort direction: want 400, got %d", code)
		}
	})

	// ── Invalid shorthand filter ────────────────────────────────────────────────

	t.Run("malformed filter returns 400", func(t *testing.T) {
		code, _ := testutil.Req(t, srv, "GET", rowsBase+"?filter=justonepart", nil)
		if code != http.StatusBadRequest {
			t.Errorf("malformed filter: want 400, got %d", code)
		}
	})
}

// TestRowUpdateDelete covers the PUT (full replace) and DELETE row endpoints.
func TestRowUpdateDelete(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")
	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "things"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")
	cb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode), map[string]any{
		"name": "title", "type": "text",
	}, http.StatusCreated)
	columnCode := testutil.Field(t, testutil.Obj(t, cb), "code")
	rowsBase := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode)

	rb := testutil.MustReq(t, srv, "POST", rowsBase, map[string]any{
		"data": map[string]any{columnCode: "original"},
	}, http.StatusCreated)
	rowID := testutil.Obj(t, rb)["id"].(float64)
	rowPath := fmt.Sprintf("%s%.0f", rowsBase, rowID)

	t.Run("PUT replaces row data", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "PUT", rowPath, map[string]any{
			"data": map[string]any{columnCode: "replaced"},
		}, http.StatusOK)
		data := testutil.Obj(t, testutil.Obj(t, b)["data"])
		if data[columnCode] != "replaced" {
			t.Errorf("data[%s] = %v, want replaced", columnCode, data[columnCode])
		}
	})

	t.Run("DELETE removes the row", func(t *testing.T) {
		testutil.MustReq(t, srv, "DELETE", rowPath, nil, http.StatusNoContent)
		testutil.MustReq(t, srv, "GET", rowPath, nil, http.StatusNotFound)
	})

	t.Run("list is empty after delete", func(t *testing.T) {
		rb := testutil.MustReq(t, srv, "GET", rowsBase, nil, http.StatusOK)
		o := testutil.Obj(t, rb)
		rows := testutil.Arr(t, o["data"])
		if len(rows) != 0 {
			t.Errorf("want 0 rows after delete, got %d", len(rows))
		}
	})
}
