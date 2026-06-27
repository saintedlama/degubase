package records_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/saintedlama/degubase/internal/testutil"
)

// TestReferencingRecords verifies that the /referencing endpoint returns
// paginated records that link to the target row.
func TestReferencingRecords(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "refws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	projb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "projects"}, http.StatusCreated)
	projCode := testutil.Field(t, testutil.Obj(t, projb), "code")

	taskb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "tasks"}, http.StatusCreated)
	taskCode := testutil.Field(t, testutil.Obj(t, taskb), "code")

	// Text column on tasks for label resolution.
	tcb := testutil.MustReq(t, srv, "POST",
		fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, taskCode),
		map[string]any{"name": "title", "type": "text"}, http.StatusCreated)
	taskTitleCode := testutil.Field(t, testutil.Obj(t, tcb), "code")

	// Row-link column on tasks → projects.
	rlcb := testutil.MustReq(t, srv, "POST",
		fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, taskCode),
		map[string]any{"name": "project", "type": "row-link", "options": map[string]any{"targetTableCode": projCode}},
		http.StatusCreated)
	rlCode := testutil.Field(t, testutil.Obj(t, rlcb), "code")

	// Create a project row.
	prb := testutil.MustReq(t, srv, "POST",
		fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, projCode),
		map[string]any{"data": map[string]any{}}, http.StatusCreated)
	projRowID := testutil.Obj(t, prb)["id"].(float64)

	// Create two tasks linking to the project.
	for i, title := range []string{"Fix login", "Add dashboard"} {
		trb := testutil.MustReq(t, srv, "POST",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, taskCode),
			map[string]any{"data": map[string]any{
				taskTitleCode: title,
				rlCode:        map[string]any{"id": projRowID, "label": "Project"},
			}}, http.StatusCreated)
		_ = testutil.Obj(t, trb)["id"].(float64)
		_ = i
	}

	// Create a task NOT linked to the project.
	testutil.MustReq(t, srv, "POST",
		fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, taskCode),
		map[string]any{"data": map[string]any{taskTitleCode: "Unrelated"}}, http.StatusCreated)

	projPath := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f/referencing", wsCode, projCode, projRowID)

	t.Run("returns referencing records", func(t *testing.T) {
		rb := testutil.MustReq(t, srv, "GET", projPath, nil, http.StatusOK)
		o := testutil.Obj(t, rb)
		rows := testutil.Arr(t, o["data"])
		if len(rows) != 2 {
			t.Fatalf("expected 2 referencing rows, got %d", len(rows))
		}
		total := o["total"].(float64)
		if total != 2 {
			t.Errorf("total: got %v, want 2", total)
		}

		// Verify first referencing row fields.
		r1 := testutil.Obj(t, rows[0])
		if r1["sourceTableCode"] != taskCode {
			t.Errorf("sourceTableCode: got %v, want %v", r1["sourceTableCode"], taskCode)
		}
		if r1["viaColumnCode"] != rlCode {
			t.Errorf("viaColumnCode: got %v, want %v", r1["viaColumnCode"], rlCode)
		}
		// Label should resolve to the task title.
		label := r1["sourceRowLabel"].(string)
		if label == "" || label == fmt.Sprintf("Row #%.0f", r1["sourceRowId"]) {
			t.Errorf("sourceRowLabel should resolve to task title, got %q", label)
		}
	})

	t.Run("non-existent row returns 404", func(t *testing.T) {
		badPath := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/99999/referencing", wsCode, projCode)
		testutil.MustReq(t, srv, "GET", badPath, nil, http.StatusNotFound)
	})

	t.Run("row with no references returns empty list", func(t *testing.T) {
		// Create an unreferenced project row.
		prb2 := testutil.MustReq(t, srv, "POST",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, projCode),
			map[string]any{"data": map[string]any{}}, http.StatusCreated)
		id2 := testutil.Obj(t, prb2)["id"].(float64)
		path := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f/referencing", wsCode, projCode, id2)

		rb := testutil.MustReq(t, srv, "GET", path, nil, http.StatusOK)
		o := testutil.Obj(t, rb)
		rows := testutil.Arr(t, o["data"])
		if len(rows) != 0 {
			t.Errorf("expected 0 rows, got %d", len(rows))
		}
	})

	t.Run("pagination works", func(t *testing.T) {
		// First page with pageSize=1.
		rb := testutil.MustReq(t, srv, "GET", projPath+"?pageSize=1&page=1", nil, http.StatusOK)
		o := testutil.Obj(t, rb)
		rows := testutil.Arr(t, o["data"])
		if len(rows) != 1 {
			t.Fatalf("page 1: expected 1 row, got %d", len(rows))
		}
		if o["total"].(float64) != 2 {
			t.Errorf("total should be 2 even when paginated")
		}

		// Second page.
		rb2 := testutil.MustReq(t, srv, "GET", projPath+"?pageSize=1&page=2", nil, http.StatusOK)
		o2 := testutil.Obj(t, rb2)
		rows2 := testutil.Arr(t, o2["data"])
		if len(rows2) != 1 {
			t.Fatalf("page 2: expected 1 row, got %d", len(rows2))
		}
	})
}
