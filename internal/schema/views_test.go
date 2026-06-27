package schema_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/saintedlama/degubase/internal/testutil"
)

func TestViews(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")
	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "t"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")
	base := fmt.Sprintf("/api/workspaces/%s/tables/%s/views/", wsCode, tblCode)

	var defaultViewCode string
	var gridViewCode string

	t.Run("list shows auto-created default view", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		views := testutil.Arr(t, b)
		if len(views) != 1 {
			t.Fatalf("expected 1 default view, got %d", len(views))
		}
		v := testutil.Obj(t, views[0])
		if testutil.Field(t, v, "type") != "tabular" {
			t.Errorf("default view type should be tabular")
		}
		defaultViewCode = testutil.Field(t, v, "code")
	})

	t.Run("create tabular view", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name": "My Grid",
			"type": "tabular",
		}, http.StatusCreated)
		o := testutil.Obj(t, b)
		gridViewCode = testutil.Field(t, o, "code")
		if testutil.Field(t, o, "name") != "My Grid" {
			t.Errorf("name mismatch")
		}
	})

	t.Run("create kanban view with config", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name":   "Board",
			"type":   "kanban",
			"config": map[string]any{"group_by_column": "status"},
		}, http.StatusCreated)
		o := testutil.Obj(t, b)
		if testutil.Field(t, o, "type") != "kanban" {
			t.Errorf("type should be kanban")
		}
		if o["config"] == nil {
			t.Errorf("config missing")
		}
	})

	t.Run("create timeline view with visibleWeekdays config", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name":   "Timeline",
			"type":   "timeline",
			"config": map[string]any{"visibleWeekdays": []any{0, 1, 2, 3, 4}},
		}, http.StatusCreated)
		o := testutil.Obj(t, b)
		cfg := testutil.Obj(t, o["config"])
		wd := testutil.Arr(t, cfg["visibleWeekdays"])
		if len(wd) != 5 {
			t.Errorf("expected 5 weekdays, got %d", len(wd))
		}
		// Verify round-trip by fetching it back
		code := testutil.Field(t, o, "code")
		b2 := testutil.MustReq(t, srv, "GET", base+code, nil, http.StatusOK)
		o2 := testutil.Obj(t, b2)
		cfg2 := testutil.Obj(t, o2["config"])
		wd2 := testutil.Arr(t, cfg2["visibleWeekdays"])
		if len(wd2) != 5 {
			t.Errorf("round-trip: expected 5 weekdays, got %d", len(wd2))
		}
	})

	t.Run("update view preserves visibleWeekdays", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name":   "Timeline Two",
			"type":   "timeline",
			"config": map[string]any{"visibleWeekdays": []any{0, 1, 2, 3, 4}},
		}, http.StatusCreated)
		o := testutil.Obj(t, b)
		code := testutil.Field(t, o, "code")
		// Update with a different set of weekdays
		b2 := testutil.MustReq(t, srv, "PUT", base+code, map[string]any{
			"name":   "Timeline Two Renamed",
			"config": map[string]any{"visibleWeekdays": []any{0, 6}},
		}, http.StatusOK)
		o2 := testutil.Obj(t, b2)
		cfg := testutil.Obj(t, o2["config"])
		wd := testutil.Arr(t, cfg["visibleWeekdays"])
		if len(wd) != 2 {
			t.Errorf("update: expected 2 weekdays, got %d", len(wd))
		}
	})

	t.Run("create kanban view with hiddenKanbanColumns config", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name":   "Board",
			"type":   "kanban",
			"config": map[string]any{"hiddenKanbanColumns": []any{"Done"}},
		}, http.StatusCreated)
		o := testutil.Obj(t, b)
		cfg := testutil.Obj(t, o["config"])
		cols := testutil.Arr(t, cfg["hiddenKanbanColumns"])
		if len(cols) != 1 {
			t.Errorf("expected 1 hidden column, got %d", len(cols))
		}
		// Round-trip
		code := testutil.Field(t, o, "code")
		b2 := testutil.MustReq(t, srv, "GET", base+code, nil, http.StatusOK)
		o2 := testutil.Obj(t, b2)
		cfg2 := testutil.Obj(t, o2["config"])
		cols2 := testutil.Arr(t, cfg2["hiddenKanbanColumns"])
		if len(cols2) != 1 {
			t.Errorf("round-trip: expected 1 hidden column, got %d", len(cols2))
		}
	})

	t.Run("update view preserves hiddenKanbanColumns", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name":   "Board Two",
			"type":   "kanban",
			"config": map[string]any{"hiddenKanbanColumns": []any{"Done"}},
		}, http.StatusCreated)
		o := testutil.Obj(t, b)
		code := testutil.Field(t, o, "code")
		b2 := testutil.MustReq(t, srv, "PUT", base+code, map[string]any{
			"name":   "Board Two Renamed",
			"config": map[string]any{"hiddenKanbanColumns": []any{"Done", "Released"}},
		}, http.StatusOK)
		o2 := testutil.Obj(t, b2)
		cfg := testutil.Obj(t, o2["config"])
		cols := testutil.Arr(t, cfg["hiddenKanbanColumns"])
		if len(cols) != 2 {
			t.Errorf("update: expected 2 hidden columns, got %d", len(cols))
		}
	})

	t.Run("create view with invalid type returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST", base, map[string]any{"name": "x", "type": "invalid"}, http.StatusBadRequest)
	})

	t.Run("create view missing name returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST", base, map[string]any{"type": "tabular"}, http.StatusBadRequest)
	})

	t.Run("list shows all views", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 7 {
			t.Errorf("expected 7 views (default + 6 created), got %d", len(testutil.Arr(t, b)))
		}
	})

	t.Run("get view by code", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base+defaultViewCode, nil, http.StatusOK)
		if testutil.Field(t, testutil.Obj(t, b), "type") != "tabular" {
			t.Errorf("type mismatch on get")
		}
	})

	t.Run("get non-existent view returns 404", func(t *testing.T) {
		testutil.MustReq(t, srv, "GET", base+"does-not-exist", nil, http.StatusNotFound)
	})

	t.Run("update view", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "PUT", base+gridViewCode, map[string]any{
			"name": "My Grid Renamed",
		}, http.StatusOK)
		if testutil.Field(t, testutil.Obj(t, b), "name") != "My Grid Renamed" {
			t.Errorf("name not updated")
		}
		gridViewCode = testutil.Field(t, testutil.Obj(t, b), "code")
	})

	t.Run("delete non-default view", func(t *testing.T) {
		testutil.MustReq(t, srv, "DELETE", base+gridViewCode, nil, http.StatusNoContent)
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 6 {
			t.Errorf("expected 6 views after delete")
		}
	})

	t.Run("cannot delete default view", func(t *testing.T) {
		code, _ := testutil.Req(t, srv, "DELETE", base+defaultViewCode, nil)
		if code == http.StatusNoContent {
			t.Errorf("deleting default view should fail")
		}
	})

	t.Run("update non-existent view returns 404", func(t *testing.T) {
		testutil.MustReq(t, srv, "PUT", base+"does-not-exist", map[string]any{"name": "x"}, http.StatusNotFound)
	})

	t.Run("update missing name returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "PUT", base+defaultViewCode, map[string]any{}, http.StatusBadRequest)
	})

	t.Run("delete non-existent view returns 404", func(t *testing.T) {
		testutil.MustReq(t, srv, "DELETE", base+"does-not-exist", nil, http.StatusNotFound)
	})
}
