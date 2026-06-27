package schema_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/saintedlama/degubase/internal/testutil"
)

func TestColumns(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")
	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "t"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")
	base := fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode)

	t.Run("list empty", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 0 {
			t.Errorf("expected empty column list")
		}
	})

	var col1Code string

	t.Run("create text column", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name": "title",
			"type": "text",
		}, http.StatusCreated)
		o := testutil.Obj(t, b)
		col1Code = testutil.Field(t, o, "code")
		if testutil.Field(t, o, "name") != "title" {
			t.Errorf("name mismatch")
		}
		if testutil.Field(t, o, "type") != "text" {
			t.Errorf("type mismatch")
		}
	})

	t.Run("create single-select column with options", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name":    "status",
			"type":    "single-select",
			"options": map[string]any{"choices": []string{"todo", "done"}},
		}, http.StatusCreated)
		o := testutil.Obj(t, b)
		if testutil.Field(t, o, "type") != "single-select" {
			t.Errorf("type mismatch")
		}
		if o["options"] == nil {
			t.Errorf("options missing")
		}
	})

	t.Run("create invalid type returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST", base, map[string]any{"name": "x", "type": "unknown-type"}, http.StatusBadRequest)
	})

	t.Run("create missing name returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST", base, map[string]any{"type": "text"}, http.StatusBadRequest)
	})

	t.Run("list shows created columns", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 2 {
			t.Errorf("expected 2 columns, got %d", len(testutil.Arr(t, b)))
		}
	})

	t.Run("update column", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "PUT", fmt.Sprintf("%s%s", base, col1Code), map[string]any{
			"name": "title-renamed",
		}, http.StatusOK)
		if testutil.Field(t, testutil.Obj(t, b), "name") != "title-renamed" {
			t.Errorf("name not updated")
		}
	})

	t.Run("reorder columns", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		cols := testutil.Arr(t, b)
		codes := make([]string, len(cols))
		for i, c := range cols {
			codes[len(cols)-1-i] = testutil.Field(t, testutil.Obj(t, c), "code")
		}
		testutil.MustReq(t, srv, "PUT", base+"reorder", map[string]any{"codes": codes}, http.StatusNoContent)
	})

	t.Run("reorder with empty codes returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "PUT", base+"reorder", map[string]any{"codes": []string{}}, http.StatusBadRequest)
	})

	t.Run("delete column", func(t *testing.T) {
		testutil.MustReq(t, srv, "DELETE", fmt.Sprintf("%s%s", base, col1Code), nil, http.StatusNoContent)
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 1 {
			t.Errorf("expected 1 column after delete")
		}
	})
}

func TestColumnValidation(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")
	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "t"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")
	base := fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode)

	cb := testutil.MustReq(t, srv, "POST", base, map[string]any{"name": "title", "type": "text"}, http.StatusCreated)
	columnCode := testutil.Field(t, testutil.Obj(t, cb), "code")
	testutil.MustReq(t, srv, "POST", base, map[string]any{"name": "score", "type": "number"}, http.StatusCreated)

	t.Run("update missing name returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "PUT", base+columnCode, map[string]any{"type": "text"}, http.StatusBadRequest)
	})

	t.Run("update non-existent column returns 404", func(t *testing.T) {
		testutil.MustReq(t, srv, "PUT", base+"does-not-exist", map[string]any{"name": "x"}, http.StatusNotFound)
	})

	t.Run("update invalid type returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "PUT", base+columnCode, map[string]any{"name": "title", "type": "bogus"}, http.StatusBadRequest)
	})

	t.Run("update cross-family type change returns 422", func(t *testing.T) {
		// text → number is an incompatible family change
		code, _ := testutil.Req(t, srv, "PUT", base+columnCode, map[string]any{"name": "title", "type": "number"})
		if code != http.StatusUnprocessableEntity {
			t.Errorf("cross-family type change: want 422, got %d", code)
		}
	})

	t.Run("delete non-existent column returns 404", func(t *testing.T) {
		testutil.MustReq(t, srv, "DELETE", base+"does-not-exist", nil, http.StatusNotFound)
	})
}
