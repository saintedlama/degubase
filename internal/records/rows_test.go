package records_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/saintedlama/degubase/internal/testutil"
)

func TestRowGetPatchHistory(t *testing.T) {
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
		"data": map[string]any{columnCode: "hello"},
	}, http.StatusCreated)
	rowID := testutil.Obj(t, rb)["id"].(float64)
	rowPath := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, tblCode, rowID)

	t.Run("get single row returns correct data", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", rowPath, nil, http.StatusOK)
		o := testutil.Obj(t, b)
		if o["id"] != rowID {
			t.Errorf("id: got %v, want %v", o["id"], rowID)
		}
		data := testutil.Obj(t, o["data"])
		if data[columnCode] != "hello" {
			t.Errorf("data[%s] = %v, want hello", columnCode, data[columnCode])
		}
	})

	t.Run("get non-existent row returns 404", func(t *testing.T) {
		testutil.MustReq(t, srv, "GET", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/99999", wsCode, tblCode), nil, http.StatusNotFound)
	})

	t.Run("patch row updates only supplied fields", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "PATCH", rowPath, map[string]any{
			"data": map[string]any{columnCode: "patched"},
		}, http.StatusOK)
		o := testutil.Obj(t, b)
		data := testutil.Obj(t, o["data"])
		if data[columnCode] != "patched" {
			t.Errorf("data[%s] = %v, want patched", columnCode, data[columnCode])
		}
	})

	t.Run("history records create and patch", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", rowPath+"/history", nil, http.StatusOK)
		entries := testutil.Arr(t, b)
		if len(entries) < 2 {
			t.Errorf("expected >= 2 history entries (create + patch), got %d", len(entries))
		}
	})
}

func TestRowSelectParam(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "selws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "things"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")

	cb1 := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode), map[string]any{"name": "title", "type": "text"}, http.StatusCreated)
	col1 := testutil.Field(t, testutil.Obj(t, cb1), "code")

	cb2 := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode), map[string]any{"name": "status", "type": "text"}, http.StatusCreated)
	col2 := testutil.Field(t, testutil.Obj(t, cb2), "code")

	rowsBase := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode)
	rb := testutil.MustReq(t, srv, "POST", rowsBase, map[string]any{
		"data": map[string]any{col1: "hello", col2: "open"},
	}, http.StatusCreated)
	rowID := testutil.Obj(t, rb)["id"].(float64)
	rowPath := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, tblCode, rowID)

	t.Run("list with select returns only requested columns", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", rowsBase+"?select="+col1, nil, http.StatusOK)
		items := testutil.Arr(t, testutil.Obj(t, b)["data"])
		if len(items) == 0 {
			t.Fatal("expected at least one row")
		}
		data := testutil.Obj(t, items[0].(map[string]any)["data"])
		if _, ok := data[col1]; !ok {
			t.Errorf("selected column %q missing from data", col1)
		}
		if _, ok := data[col2]; ok {
			t.Errorf("non-selected column %q should be absent from data", col2)
		}
	})

	t.Run("get with select returns only requested columns", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", rowPath+"?select="+col2, nil, http.StatusOK)
		data := testutil.Obj(t, testutil.Obj(t, b)["data"])
		if _, ok := data[col2]; !ok {
			t.Errorf("selected column %q missing from data", col2)
		}
		if _, ok := data[col1]; ok {
			t.Errorf("non-selected column %q should be absent from data", col1)
		}
	})

	t.Run("no select returns all columns", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", rowPath, nil, http.StatusOK)
		data := testutil.Obj(t, testutil.Obj(t, b)["data"])
		if _, ok := data[col1]; !ok {
			t.Errorf("column %q missing without select", col1)
		}
		if _, ok := data[col2]; !ok {
			t.Errorf("column %q missing without select", col2)
		}
	})
}

func TestAnnotations(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "annws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "things"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")

	rowsBase := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode)
	rb := testutil.MustReq(t, srv, "POST", rowsBase, map[string]any{"data": map[string]any{}}, http.StatusCreated)
	rowID := testutil.Obj(t, rb)["id"].(float64)
	histBase := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f/history", wsCode, tblCode, rowID)

	var annotationID float64

	t.Run("create annotation returns 201 with entry_type=annotation", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", histBase, map[string]any{"annotation": "initial note"}, http.StatusCreated)
		o := testutil.Obj(t, b)
		if o["entry_type"] != "annotation" {
			t.Errorf("entry_type: got %v, want annotation", o["entry_type"])
		}
		if o["annotation"] != "initial note" {
			t.Errorf("annotation: got %v, want 'initial note'", o["annotation"])
		}
		annotationID = o["id"].(float64)
	})

	t.Run("list history includes annotation entry", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", histBase, nil, http.StatusOK)
		entries := testutil.Arr(t, b)
		found := false
		for _, e := range entries {
			m := e.(map[string]any)
			if m["entry_type"] == "annotation" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected an annotation entry in history list")
		}
	})

	t.Run("patch annotation updates text", func(t *testing.T) {
		path := fmt.Sprintf("%s/%.0f", histBase, annotationID)
		b := testutil.MustReq(t, srv, "PATCH", path, map[string]any{"annotation": "updated note"}, http.StatusOK)
		o := testutil.Obj(t, b)
		if o["annotation"] != "updated note" {
			t.Errorf("annotation: got %v, want 'updated note'", o["annotation"])
		}
	})

	t.Run("patch change entry annotation adds a note to change entry", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", histBase, nil, http.StatusOK)
		entries := testutil.Arr(t, b)
		var changeID float64
		for _, e := range entries {
			m := e.(map[string]any)
			if m["entry_type"] == "change" {
				changeID = m["id"].(float64)
				break
			}
		}
		if changeID == 0 {
			t.Skip("no change entry found")
		}
		path := fmt.Sprintf("%s/%.0f", histBase, changeID)
		b2 := testutil.MustReq(t, srv, "PATCH", path, map[string]any{"annotation": "row created here"}, http.StatusOK)
		o := testutil.Obj(t, b2)
		if o["annotation"] != "row created here" {
			t.Errorf("annotation: got %v, want 'row created here'", o["annotation"])
		}
	})

	t.Run("delete annotation returns 204 and removes entry", func(t *testing.T) {
		path := fmt.Sprintf("%s/%.0f", histBase, annotationID)
		testutil.MustReq(t, srv, "DELETE", path, nil, http.StatusNoContent)

		b := testutil.MustReq(t, srv, "GET", histBase, nil, http.StatusOK)
		entries := testutil.Arr(t, b)
		for _, e := range entries {
			m := e.(map[string]any)
			if m["id"].(float64) == annotationID {
				t.Error("deleted annotation still present in history")
			}
		}
	})

	t.Run("create annotation with empty text returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST", histBase, map[string]any{"annotation": ""}, http.StatusBadRequest)
	})
}
