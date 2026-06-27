package schema_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/saintedlama/degubase/internal/testutil"
)

func TestTables(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")
	base := fmt.Sprintf("/api/workspaces/%s/tables/", wsCode)

	t.Run("list empty", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 0 {
			t.Errorf("expected empty table list")
		}
	})

	var tblCode string

	t.Run("create", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name":    "projects",
			"context": "all projects",
		}, http.StatusCreated)
		o := testutil.Obj(t, b)
		tblCode = testutil.Field(t, o, "code")
		if tblCode == "" {
			t.Errorf("code should not be empty")
		}
		if testutil.Field(t, o, "name") != "projects" {
			t.Errorf("name mismatch")
		}
		views := testutil.Arr(t, o["views"])
		if len(views) != 1 {
			t.Fatalf("expected 1 default view, got %d", len(views))
		}
		v := testutil.Obj(t, views[0])
		if testutil.Field(t, v, "type") != "tabular" {
			t.Errorf("default view should be tabular")
		}
		if o["default_view_id"] == nil {
			t.Errorf("table should have default_view_id set")
		}
		if testutil.Field(t, v, "code") != "default-view" {
			t.Errorf("default view code want default-view, got %s", testutil.Field(t, v, "code"))
		}
	})

	t.Run("create resolves circular FK: table ↔ default view", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name": "circular-test",
		}, http.StatusCreated)
		o := testutil.Obj(t, b)

		// Table has default_view_id set
		dvID, ok := o["default_view_id"]
		if !ok || dvID == nil {
			t.Fatalf("table must have default_view_id set")
		}
		dvIDNum := int64(dvID.(float64))

		// That default_view_id matches an actual view
		views := testutil.Arr(t, o["views"])
		found := false
		for _, raw := range views {
			vo := testutil.Obj(t, raw)
			if int64(vo["id"].(float64)) == dvIDNum {
				found = true
				// View belongs to this table
				if int64(vo["table_id"].(float64)) != int64(o["id"].(float64)) {
					t.Errorf("default view's table_id should match table id")
				}
				break
			}
		}
		if !found {
			t.Errorf("default_view_id (%d) not found in embedded views", dvIDNum)
		}
	})

	t.Run("create missing name returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST", base, map[string]any{"context": "x"}, http.StatusBadRequest)
	})

	t.Run("list after create", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) < 1 {
			t.Errorf("expected at least 1 table")
		}
	})

	t.Run("get embeds views and columns", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base+tblCode+"/", nil, http.StatusOK)
		o := testutil.Obj(t, b)
		if o["views"] == nil {
			t.Errorf("views not embedded")
		}
		if o["columns"] == nil {
			t.Errorf("columns not embedded")
		}
	})

	t.Run("get non-existent returns 404", func(t *testing.T) {
		testutil.MustReq(t, srv, "GET", base+"does-not-exist/", nil, http.StatusNotFound)
	})

	t.Run("update", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "PUT", base+tblCode+"/", map[string]any{
			"name":    "projects-v2",
			"context": "updated",
		}, http.StatusOK)
		o := testutil.Obj(t, b)
		if testutil.Field(t, o, "name") != "projects-v2" {
			t.Errorf("name not updated")
		}
		tblCode = testutil.Field(t, o, "code")
	})

	t.Run("delete", func(t *testing.T) {
		testutil.MustReq(t, srv, "DELETE", base+tblCode+"/", nil, http.StatusNoContent)
		testutil.MustReq(t, srv, "GET", base+tblCode+"/", nil, http.StatusNotFound)
	})
}

func TestTableValidation(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")
	base := fmt.Sprintf("/api/workspaces/%s/tables/", wsCode)

	tb := testutil.MustReq(t, srv, "POST", base, map[string]any{"name": "t"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")

	t.Run("update missing name returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "PUT", base+tblCode+"/", map[string]any{"context": "only context"}, http.StatusBadRequest)
	})

	t.Run("workspace not found returns 404", func(t *testing.T) {
		testutil.MustReq(t, srv, "GET", "/api/workspaces/does-not-exist/tables/", nil, http.StatusNotFound)
	})
}

func TestTableIcon(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws-icon"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")
	base := fmt.Sprintf("/api/workspaces/%s/tables/", wsCode)

	t.Run("create with icon", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name": "icon-table",
			"icon": "ri-database-2-line",
		}, http.StatusCreated)
		o := testutil.Obj(t, b)
		if testutil.Field(t, o, "icon") != "ri-database-2-line" {
			t.Errorf("icon want ri-database-2-line, got %s", testutil.Field(t, o, "icon"))
		}
		tblCode := testutil.Field(t, o, "code")

		// fetch and verify icon is returned
		b2 := testutil.MustReq(t, srv, "GET", base+tblCode+"/", nil, http.StatusOK)
		o2 := testutil.Obj(t, b2)
		if testutil.Field(t, o2, "icon") != "ri-database-2-line" {
			t.Errorf("fetched icon want ri-database-2-line, got %s", testutil.Field(t, o2, "icon"))
		}

		// update icon
		b3 := testutil.MustReq(t, srv, "PUT", base+tblCode+"/", map[string]any{
			"name": "icon-table",
			"icon": "ri-table-line",
		}, http.StatusOK)
		o3 := testutil.Obj(t, b3)
		if testutil.Field(t, o3, "icon") != "ri-table-line" {
			t.Errorf("updated icon want ri-table-line, got %s", testutil.Field(t, o3, "icon"))
		}
	})
}
