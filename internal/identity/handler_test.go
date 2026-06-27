package identity_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/saintedlama/degubase/internal/testutil"
)

func TestWorkspaces(t *testing.T) {
	srv := testutil.NewServer(t)
	base := "/api/workspaces/"

	t.Run("list empty", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 0 {
			t.Errorf("expected empty list")
		}
	})

	var wsCode string

	t.Run("create", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name":    "engineering",
			"context": "all eng work",
		}, http.StatusCreated)
		o := testutil.Obj(t, b)
		wsCode = testutil.Field(t, o, "code")
		if wsCode == "" {
			t.Errorf("code should not be empty")
		}
		if testutil.Field(t, o, "name") != "engineering" {
			t.Errorf("name mismatch")
		}
		if testutil.Field(t, o, "context") != "all eng work" {
			t.Errorf("context mismatch")
		}
		if o["id"] == nil {
			t.Errorf("id missing")
		}
		if o["created_at"] == nil {
			t.Errorf("created_at missing")
		}
	})

	t.Run("create missing name returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST", base, map[string]any{"context": "no name"}, http.StatusBadRequest)
	})

	t.Run("list after create", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 1 {
			t.Errorf("expected 1 workspace")
		}
	})

	t.Run("get", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base+wsCode+"/", nil, http.StatusOK)
		if testutil.Field(t, testutil.Obj(t, b), "name") != "engineering" {
			t.Errorf("name mismatch on get")
		}
	})

	t.Run("get non-existent returns 404", func(t *testing.T) {
		testutil.MustReq(t, srv, "GET", base+"does-not-exist/", nil, http.StatusNotFound)
	})

	t.Run("update", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "PUT", base+wsCode+"/", map[string]any{
			"name":    "engineering-updated",
			"context": "updated ctx",
		}, http.StatusOK)
		o := testutil.Obj(t, b)
		if testutil.Field(t, o, "name") != "engineering-updated" {
			t.Errorf("name not updated")
		}
		if testutil.Field(t, o, "context") != "updated ctx" {
			t.Errorf("context not updated")
		}
		wsCode = testutil.Field(t, o, "code")
	})

	t.Run("update missing name returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "PUT", base+wsCode+"/", map[string]any{"context": "x"}, http.StatusBadRequest)
	})

	t.Run("delete", func(t *testing.T) {
		testutil.MustReq(t, srv, "DELETE", base+wsCode+"/", nil, http.StatusNoContent)
		testutil.MustReq(t, srv, "GET", base+wsCode+"/", nil, http.StatusNotFound)
	})

	t.Run("list after delete is empty", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 0 {
			t.Errorf("expected empty list after delete")
		}
	})

	t.Run("deleting workspace cascades to tables and rows", func(t *testing.T) {
		wb := testutil.MustReq(t, srv, "POST", base, map[string]any{"name": "cascade-ws"}, http.StatusCreated)
		wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

		tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "t"}, http.StatusCreated)
		tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")
		rowBase := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode)

		testutil.MustReq(t, srv, "POST", rowBase, map[string]any{"data": map[string]any{"x": 1}}, http.StatusCreated)
		testutil.MustReq(t, srv, "POST", rowBase, map[string]any{"data": map[string]any{"x": 2}}, http.StatusCreated)

		testutil.MustReq(t, srv, "DELETE", fmt.Sprintf("/api/workspaces/%s/", wsCode), nil, http.StatusNoContent)
		testutil.MustReq(t, srv, "GET", fmt.Sprintf("/api/workspaces/%s/tables/%s/", wsCode, tblCode), nil, http.StatusNotFound)
	})
}
