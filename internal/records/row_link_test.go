package records_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/saintedlama/degubase/internal/testutil"
)

// TestRowLinkColumn verifies that a row-link column can be created with the
// correct type and that its targettableCode option is persisted and returned.
func TestRowLinkColumn(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "colws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "things"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")

	tb2 := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "other"}, http.StatusCreated)
	tbl2Code := testutil.Field(t, testutil.Obj(t, tb2), "code") // used as targettableCode

	t.Run("create cross-table row-link column", func(t *testing.T) {
		cb := testutil.MustReq(t, srv, "POST",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode),
			map[string]any{"name": "linked", "type": "row-link", "options": map[string]any{"targetTableCode": tbl2Code}},
			http.StatusCreated)
		col := testutil.Obj(t, cb)
		if testutil.Field(t, col, "type") != "row-link" {
			t.Errorf("type: got %v, want row-link", col["type"])
		}
		opts, ok := col["options"].(map[string]any)
		if !ok || opts == nil {
			t.Fatal("options missing from response")
		}
		if opts["targetTableCode"] != tbl2Code {
			t.Errorf("targettableCode: got %v, want %v", opts["targetTableCode"], tbl2Code)
		}
	})

	t.Run("create same-table row-link column (hierarchy)", func(t *testing.T) {
		cb := testutil.MustReq(t, srv, "POST",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode),
			map[string]any{"name": "parent", "type": "row-link", "options": map[string]any{"targetTableCode": tblCode}},
			http.StatusCreated)
		col := testutil.Obj(t, cb)
		if testutil.Field(t, col, "type") != "row-link" {
			t.Errorf("type: got %v, want row-link", col["type"])
		}
		opts := testutil.Obj(t, col["options"])
		if opts["targetTableCode"] != tblCode {
			t.Errorf("targettableCode: got %v, want %v (same table)", opts["targetTableCode"], tblCode)
		}
	})

	t.Run("invalid type rejected", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode),
			map[string]any{"name": "bad", "type": "not-a-real-type"},
			http.StatusBadRequest)
	})
}

// TestRowLinkValues covers storing, retrieving, updating, and clearing a
// row-link cell value.
func TestRowLinkValues(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "rlvws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	projb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "projects"}, http.StatusCreated)
	projCode := testutil.Field(t, testutil.Obj(t, projb), "code")

	taskb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "tasks"}, http.StatusCreated)
	taskCode := testutil.Field(t, testutil.Obj(t, taskb), "code")

	// name column on projects — the resolver uses this as the label source
	ncb := testutil.MustReq(t, srv, "POST",
		fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, projCode),
		map[string]any{"name": "name", "type": "text"}, http.StatusCreated)
	projNameCode := testutil.Field(t, testutil.Obj(t, ncb), "code")

	rlcb := testutil.MustReq(t, srv, "POST",
		fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, taskCode),
		map[string]any{"name": "project", "type": "row-link", "options": map[string]any{"targetTableCode": projCode}},
		http.StatusCreated)
	rlcolumnCode := testutil.Field(t, testutil.Obj(t, rlcb), "code")

	// Create a project row with name "Alpha"
	prb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, projCode),
		map[string]any{"data": map[string]any{projNameCode: "Alpha"}}, http.StatusCreated)
	projRowID := testutil.Obj(t, prb)["id"].(float64)

	t.Run("store and retrieve link", func(t *testing.T) {
		rb := testutil.MustReq(t, srv, "POST",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, taskCode),
			map[string]any{"data": map[string]any{
				rlcolumnCode: map[string]any{"id": projRowID, "label": "Alpha"},
			}}, http.StatusCreated)
		rowID := testutil.Obj(t, rb)["id"].(float64)

		gb := testutil.MustReq(t, srv, "GET",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, taskCode, rowID),
			nil, http.StatusOK)
		data := testutil.Obj(t, testutil.Obj(t, gb)["data"])
		link, ok := data[rlcolumnCode].(map[string]any)
		if !ok {
			t.Fatalf("link value: got %T %v, want object", data[rlcolumnCode], data[rlcolumnCode])
		}
		if link["id"] != projRowID {
			t.Errorf("link.id: got %v, want %v", link["id"], projRowID)
		}
		if link["label"] != "Alpha" {
			t.Errorf("link.label: got %v, want Alpha", link["label"])
		}
	})

	t.Run("update link to different row", func(t *testing.T) {
		prb2 := testutil.MustReq(t, srv, "POST",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, projCode),
			map[string]any{"data": map[string]any{projNameCode: "Beta"}}, http.StatusCreated)
		projRowID2 := testutil.Obj(t, prb2)["id"].(float64)

		rb := testutil.MustReq(t, srv, "POST",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, taskCode),
			map[string]any{"data": map[string]any{rlcolumnCode: map[string]any{"id": projRowID, "label": "Alpha"}}},
			http.StatusCreated)
		rowID := testutil.Obj(t, rb)["id"].(float64)
		rowPath := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, taskCode, rowID)

		testutil.MustReq(t, srv, "PATCH", rowPath,
			map[string]any{"data": map[string]any{rlcolumnCode: map[string]any{"id": projRowID2, "label": "Beta"}}},
			http.StatusOK)

		gb := testutil.MustReq(t, srv, "GET", rowPath, nil, http.StatusOK)
		link := testutil.Obj(t, testutil.Obj(t, testutil.Obj(t, gb)["data"])[rlcolumnCode])
		if link["id"] != projRowID2 {
			t.Errorf("updated link.id: got %v, want %v", link["id"], projRowID2)
		}
		if link["label"] != "Beta" {
			t.Errorf("updated link.label: got %v, want Beta", link["label"])
		}
	})

	t.Run("clear link to null", func(t *testing.T) {
		rb := testutil.MustReq(t, srv, "POST",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, taskCode),
			map[string]any{"data": map[string]any{rlcolumnCode: map[string]any{"id": projRowID, "label": "Alpha"}}},
			http.StatusCreated)
		rowID := testutil.Obj(t, rb)["id"].(float64)
		rowPath := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, taskCode, rowID)

		testutil.MustReq(t, srv, "PATCH", rowPath,
			map[string]any{"data": map[string]any{rlcolumnCode: nil}},
			http.StatusOK)

		gb := testutil.MustReq(t, srv, "GET", rowPath, nil, http.StatusOK)
		data := testutil.Obj(t, testutil.Obj(t, gb)["data"])
		if data[rlcolumnCode] != nil {
			t.Errorf("cleared link: got %v, want nil", data[rlcolumnCode])
		}
	})

	t.Run("row with no link returns null", func(t *testing.T) {
		rb := testutil.MustReq(t, srv, "POST",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, taskCode),
			map[string]any{"data": map[string]any{}}, http.StatusCreated)
		rowID := testutil.Obj(t, rb)["id"].(float64)

		gb := testutil.MustReq(t, srv, "GET",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, taskCode, rowID),
			nil, http.StatusOK)
		data := testutil.Obj(t, testutil.Obj(t, gb)["data"])
		if data[rlcolumnCode] != nil {
			t.Errorf("unset link: got %v, want nil", data[rlcolumnCode])
		}
	})
}

// TestRowLinkOrphanOnDelete verifies that deleting a referenced row is blocked
// with 409 Conflict — the row and its references are left intact.
func TestRowLinkOrphanOnDelete(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "orphanws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	projb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "projects"}, http.StatusCreated)
	projCode := testutil.Field(t, testutil.Obj(t, projb), "code")

	taskb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "tasks"}, http.StatusCreated)
	taskCode := testutil.Field(t, testutil.Obj(t, taskb), "code")

	rlcb := testutil.MustReq(t, srv, "POST",
		fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, taskCode),
		map[string]any{"name": "project", "type": "row-link", "options": map[string]any{"targetTableCode": projCode}},
		http.StatusCreated)
	rlcolumnCode := testutil.Field(t, testutil.Obj(t, rlcb), "code")

	// Create the project row that will be deleted
	prb := testutil.MustReq(t, srv, "POST",
		fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, projCode),
		map[string]any{"data": map[string]any{}}, http.StatusCreated)
	ephemeralID := testutil.Obj(t, prb)["id"].(float64)
	projRowPath := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, projCode, ephemeralID)

	// Create a task linked to it
	taskRb := testutil.MustReq(t, srv, "POST",
		fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, taskCode),
		map[string]any{"data": map[string]any{
			rlcolumnCode: map[string]any{"id": ephemeralID, "label": "Ephemeral Project"},
		}}, http.StatusCreated)
	taskID := testutil.Obj(t, taskRb)["id"].(float64)
	taskPath := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, taskCode, taskID)

	// Sanity: project row exists before deletion
	testutil.MustReq(t, srv, "GET", projRowPath, nil, http.StatusOK)

	// Deleting the referenced project row must be blocked with 409.
	testutil.MustReq(t, srv, "DELETE", projRowPath, nil, http.StatusConflict)

	// The project row must still exist.
	testutil.MustReq(t, srv, "GET", projRowPath, nil, http.StatusOK)

	// The task row must still hold its link intact.
	gb := testutil.MustReq(t, srv, "GET", taskPath, nil, http.StatusOK)
	data := testutil.Obj(t, testutil.Obj(t, gb)["data"])

	if _, exists := data[rlcolumnCode]; !exists {
		t.Error("task row link must still be set after blocked delete")
	}

	// Listing the tasks table must also succeed and include the row
	lb := testutil.MustReq(t, srv, "GET",
		fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, taskCode),
		nil, http.StatusOK)
	rows := testutil.Arr(t, testutil.Obj(t, lb)["data"])
	found := false
	for _, r := range rows {
		if row, ok := r.(map[string]any); ok {
			if row["id"] == taskID {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("task row not found in list after referenced row deleted")
	}
}

// TestRowLinkSameTable covers the hierarchy (parent/child within one table)
// use case, including orphaned self-references when a parent row is deleted.
func TestRowLinkSameTable(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "hierws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "items"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")
	tableCode := testutil.Field(t, testutil.Obj(t, tb), "code") // used as targettableCode

	// Self-referencing parent column
	pcb := testutil.MustReq(t, srv, "POST",
		fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode),
		map[string]any{"name": "parent", "type": "row-link", "options": map[string]any{"targetTableCode": tableCode}},
		http.StatusCreated)
	parentcolumnCode := testutil.Field(t, testutil.Obj(t, pcb), "code")

	// Create parent row
	prb := testutil.MustReq(t, srv, "POST",
		fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode),
		map[string]any{"data": map[string]any{}}, http.StatusCreated)
	parentID := testutil.Obj(t, prb)["id"].(float64)
	parentPath := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, tblCode, parentID)

	t.Run("child stores parent link correctly", func(t *testing.T) {
		crb := testutil.MustReq(t, srv, "POST",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode),
			map[string]any{"data": map[string]any{
				parentcolumnCode: map[string]any{"id": parentID, "label": "Root"},
			}}, http.StatusCreated)
		childID := testutil.Obj(t, crb)["id"].(float64)

		gb := testutil.MustReq(t, srv, "GET",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, tblCode, childID),
			nil, http.StatusOK)
		link := testutil.Obj(t, testutil.Obj(t, testutil.Obj(t, gb)["data"])[parentcolumnCode])
		if link["id"] != parentID {
			t.Errorf("parent link.id: got %v, want %v", link["id"], parentID)
		}
	})

	t.Run("delete parent is blocked when child references it", func(t *testing.T) {
		crb := testutil.MustReq(t, srv, "POST",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode),
			map[string]any{"data": map[string]any{
				parentcolumnCode: map[string]any{"id": parentID, "label": "Root"},
			}}, http.StatusCreated)
		childID := testutil.Obj(t, crb)["id"].(float64)
		childPath := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, tblCode, childID)

		// Deleting the parent must be blocked because the child references it.
		testutil.MustReq(t, srv, "DELETE", parentPath, nil, http.StatusConflict)

		// Child link is unchanged.
		gb := testutil.MustReq(t, srv, "GET", childPath, nil, http.StatusOK)
		data := testutil.Obj(t, testutil.Obj(t, gb)["data"])
		if _, exists := data[parentcolumnCode]; !exists {
			t.Error("child link must still be set after blocked delete")
		}
	})

	t.Run("circular parent reference is rejected", func(t *testing.T) {
		// A → parent B, then B → parent A must be rejected.
		ab := testutil.MustReq(t, srv, "POST",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode),
			map[string]any{"data": map[string]any{}}, http.StatusCreated)
		aID := testutil.Obj(t, ab)["id"].(float64)

		bb := testutil.MustReq(t, srv, "POST",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode),
			map[string]any{"data": map[string]any{
				parentcolumnCode: map[string]any{"id": aID, "label": "A"},
			}}, http.StatusCreated)
		bID := testutil.Obj(t, bb)["id"].(float64)

		// Setting A's parent to B would create a cycle: A→B→A
		testutil.MustReq(t, srv, "PATCH",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, tblCode, aID),
			map[string]any{"data": map[string]any{
				parentcolumnCode: map[string]any{"id": bID, "label": "B"},
			}}, http.StatusBadRequest)

		// Setting a row as its own parent is also a cycle.
		testutil.MustReq(t, srv, "PATCH",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, tblCode, aID),
			map[string]any{"data": map[string]any{
				parentcolumnCode: map[string]any{"id": aID, "label": "A"},
			}}, http.StatusBadRequest)
	})
}
