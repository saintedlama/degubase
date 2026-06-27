package records_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/saintedlama/degubase/internal/testutil"
)

// TestRowLinkStaleLabelResolution verifies that when a linked row's label
// column is updated, subsequent reads of the linking row reflect the new
// label — even though the stored {id, label} blob was written before the
// change.
func TestRowLinkStaleLabelResolution(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "stalews"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	projb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "projects"}, http.StatusCreated)
	projCode := testutil.Field(t, testutil.Obj(t, projb), "code")

	taskb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "tasks"}, http.StatusCreated)
	taskCode := testutil.Field(t, testutil.Obj(t, taskb), "code")

	// Text "name" column on projects — this is what becomes the label.
	pcb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, projCode),
		map[string]any{"name": "name", "type": "text"}, http.StatusCreated)
	projNameCode := testutil.Field(t, testutil.Obj(t, pcb), "code")

	// Row-link column on tasks → projects.
	rlcb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, taskCode),
		map[string]any{"name": "project", "type": "row-link", "options": map[string]any{"targetTableCode": projCode}},
		http.StatusCreated)
	rlCode := testutil.Field(t, testutil.Obj(t, rlcb), "code")

	// Create a project row with name "Alpha".
	prb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, projCode),
		map[string]any{"data": map[string]any{projNameCode: "Alpha"}}, http.StatusCreated)
	projRowID := testutil.Obj(t, prb)["id"].(float64)
	projPath := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, projCode, projRowID)

	// Create a task linked to the project; label written as "Alpha".
	trb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, taskCode),
		map[string]any{"data": map[string]any{rlCode: map[string]any{"id": projRowID, "label": "Alpha"}}},
		http.StatusCreated)
	taskRowID := testutil.Obj(t, trb)["id"].(float64)
	taskPath := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, taskCode, taskRowID)

	// Sanity: task currently shows "Alpha".
	gb := testutil.MustReq(t, srv, "GET", taskPath, nil, http.StatusOK)
	link := testutil.Obj(t, testutil.Obj(t, testutil.Obj(t, gb)["data"])[rlCode])
	if link["label"] != "Alpha" {
		t.Fatalf("initial label: got %v, want Alpha", link["label"])
	}

	// Rename the project to "Beta" (changes the source-of-truth label).
	testutil.MustReq(t, srv, "PATCH", projPath,
		map[string]any{"data": map[string]any{projNameCode: "Beta"}}, http.StatusOK)

	// Single-row GET must now return "Beta" even though the stored blob still says "Alpha".
	t.Run("GET single row reflects fresh label", func(t *testing.T) {
		gb := testutil.MustReq(t, srv, "GET", taskPath, nil, http.StatusOK)
		link := testutil.Obj(t, testutil.Obj(t, testutil.Obj(t, gb)["data"])[rlCode])
		if link["label"] != "Beta" {
			t.Errorf("single GET label: got %v, want Beta", link["label"])
		}
	})

	// List rows must also return "Beta".
	t.Run("LIST rows reflects fresh label", func(t *testing.T) {
		lb := testutil.MustReq(t, srv, "GET",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, taskCode),
			nil, http.StatusOK)
		rows := testutil.Arr(t, testutil.Obj(t, lb)["data"])
		if len(rows) == 0 {
			t.Fatal("no rows returned")
		}
		rowData := testutil.Obj(t, testutil.Obj(t, rows[0])["data"])
		link := testutil.Obj(t, rowData[rlCode])
		if link["label"] != "Beta" {
			t.Errorf("list label: got %v, want Beta", link["label"])
		}
	})
}

// TestRowLinkDeleteBlocked verifies that deleting a referenced row is rejected
// with 409 Conflict and a response body listing the referencing records.
func TestRowLinkDeleteBlocked(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "staleorp"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	projb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "projects"}, http.StatusCreated)
	projCode := testutil.Field(t, testutil.Obj(t, projb), "code")

	taskb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "tasks"}, http.StatusCreated)
	taskCode := testutil.Field(t, testutil.Obj(t, taskb), "code")

	rlcb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, taskCode),
		map[string]any{"name": "project", "type": "row-link", "options": map[string]any{"targetTableCode": projCode}},
		http.StatusCreated)
	rlCode := testutil.Field(t, testutil.Obj(t, rlcb), "code")

	prb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, projCode),
		map[string]any{"data": map[string]any{}}, http.StatusCreated)
	projRowID := testutil.Obj(t, prb)["id"].(float64)
	projPath := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, projCode, projRowID)

	trb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, taskCode),
		map[string]any{"data": map[string]any{rlCode: map[string]any{"id": projRowID, "label": "Ghost Project"}}},
		http.StatusCreated)
	taskRowID := testutil.Obj(t, trb)["id"].(float64)
	taskPath := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, taskCode, taskRowID)

	// Deleting the referenced project row must be blocked with 409.
	body := testutil.MustReq(t, srv, "DELETE", projPath, nil, http.StatusConflict)
	resp := testutil.Obj(t, body)
	if resp["code"] != "referenced" {
		t.Errorf("conflict code: got %v, want 'referenced'", resp["code"])
	}
	refs, ok := resp["references"].([]any)
	if !ok || len(refs) == 0 {
		t.Fatalf("references: got %v, want a non-empty list", resp["references"])
	}
	ref := testutil.Obj(t, refs[0])
	if ref["tableCode"] != taskCode {
		t.Errorf("ref tableCode: got %v, want %v", ref["tableCode"], taskCode)
	}
	if ref["rowId"] != taskRowID {
		t.Errorf("ref rowId: got %v, want %v", ref["rowId"], taskRowID)
	}

	// The project row must still exist.
	testutil.MustReq(t, srv, "GET", projPath, nil, http.StatusOK)

	// The task row must still hold its link.
	gb := testutil.MustReq(t, srv, "GET", taskPath, nil, http.StatusOK)
	data := testutil.Obj(t, testutil.Obj(t, gb)["data"])
	if _, exists := data[rlCode]; !exists {
		t.Error("task row link must still be set after blocked delete")
	}
}
