package records_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/saintedlama/degubase/internal/testutil"
)

func TestGraphIsolatedRow(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "graphiso"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "items"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")
	tableCode := testutil.Field(t, testutil.Obj(t, tb), "code") // used in graph node IDs

	rb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode), map[string]any{"data": map[string]any{}}, http.StatusCreated)
	rowID := testutil.Obj(t, rb)["id"].(float64)

	gb := testutil.MustReq(t, srv, "GET",
		fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f/graph", wsCode, tblCode, rowID),
		nil, http.StatusOK)

	g := testutil.Obj(t, gb)
	nodes := testutil.Arr(t, g["nodes"])
	edges := testutil.Arr(t, g["edges"])

	if len(nodes) != 1 {
		t.Errorf("isolated row: expected 1 node, got %d", len(nodes))
	}
	if len(edges) != 0 {
		t.Errorf("isolated row: expected 0 edges, got %d", len(edges))
	}

	wantRoot := fmt.Sprintf("%s:%.0f", tableCode, rowID)
	if g["root"] != wantRoot {
		t.Errorf("root: got %v, want %s", g["root"], wantRoot)
	}
	node := testutil.Obj(t, nodes[0])
	if node["id"] != wantRoot {
		t.Errorf("node.id: got %v, want %s", node["id"], wantRoot)
	}
}

func TestGraphLinkedRows(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "graphlnk"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	projb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "projects"}, http.StatusCreated)
	projCode := testutil.Field(t, testutil.Obj(t, projb), "code")

	taskb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "tasks"}, http.StatusCreated)
	taskCode := testutil.Field(t, testutil.Obj(t, taskb), "code")

	pcb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, projCode), map[string]any{"name": "name", "type": "text"}, http.StatusCreated)
	projNameCol := testutil.Field(t, testutil.Obj(t, pcb), "code")

	rlcb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, taskCode), map[string]any{
		"name": "project", "type": "row-link", "options": map[string]any{"targetTableCode": projCode},
	}, http.StatusCreated)
	rlcolumnCode := testutil.Field(t, testutil.Obj(t, rlcb), "code")

	prb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, projCode),
		map[string]any{"data": map[string]any{projNameCol: "Alpha Project"}}, http.StatusCreated)
	projRowID := testutil.Obj(t, prb)["id"].(float64)

	trb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, taskCode),
		map[string]any{"data": map[string]any{rlcolumnCode: map[string]any{"id": projRowID, "label": "Alpha Project"}}},
		http.StatusCreated)
	taskRowID := testutil.Obj(t, trb)["id"].(float64)

	gb := testutil.MustReq(t, srv, "GET",
		fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f/graph", wsCode, taskCode, taskRowID),
		nil, http.StatusOK)

	g := testutil.Obj(t, gb)
	nodes := testutil.Arr(t, g["nodes"])
	edges := testutil.Arr(t, g["edges"])

	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes (task + project), got %d", len(nodes))
	}
	if len(edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(edges))
	}

	edge := testutil.Obj(t, edges[0])
	wantFrom := fmt.Sprintf("%s:%.0f", taskCode, taskRowID)
	wantTo := fmt.Sprintf("%s:%.0f", projCode, projRowID)
	if edge["from"] != wantFrom {
		t.Errorf("edge.from: got %v, want %s", edge["from"], wantFrom)
	}
	if edge["to"] != wantTo {
		t.Errorf("edge.to: got %v, want %s", edge["to"], wantTo)
	}
	if edge["via_column"] != rlcolumnCode {
		t.Errorf("edge.via_column: got %v, want %s", edge["via_column"], rlcolumnCode)
	}
	if edge["via_column_name"] != "project" {
		t.Errorf("edge.via_column_name: got %v, want 'project'", edge["via_column_name"])
	}

	// Project node label must be derived from the text column
	wantProjID := wantTo
	var projNode map[string]any
	for _, n := range nodes {
		nd := testutil.Obj(t, n)
		if nd["id"] == wantProjID {
			projNode = nd
		}
	}
	if projNode == nil {
		t.Fatal("project node not found")
	}
	if projNode["label"] != "Alpha Project" {
		t.Errorf("project node label: got %v, want 'Alpha Project'", projNode["label"])
	}
}

func TestGraphCycleDetection(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "graphcyc"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "items"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")
	tableCode := testutil.Field(t, testutil.Obj(t, tb), "code") // used in graph node IDs and targettableCode

	rlcb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode), map[string]any{
		"name": "parent", "type": "row-link", "options": map[string]any{"targetTableCode": tableCode},
	}, http.StatusCreated)
	rlcolumnCode := testutil.Field(t, testutil.Obj(t, rlcb), "code")

	// Create A, B, C then wire A→B→C→A (cycle)
	ra := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode), map[string]any{"data": map[string]any{}}, http.StatusCreated)
	idA := testutil.Obj(t, ra)["id"].(float64)
	rb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode), map[string]any{"data": map[string]any{}}, http.StatusCreated)
	idB := testutil.Obj(t, rb)["id"].(float64)
	rc := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode), map[string]any{"data": map[string]any{}}, http.StatusCreated)
	idC := testutil.Obj(t, rc)["id"].(float64)

	rowPath := func(id float64) string {
		return fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, tblCode, id)
	}
	testutil.MustReq(t, srv, "PATCH", rowPath(idA), map[string]any{"data": map[string]any{rlcolumnCode: map[string]any{"id": idB, "label": "B"}}}, http.StatusOK)
	testutil.MustReq(t, srv, "PATCH", rowPath(idB), map[string]any{"data": map[string]any{rlcolumnCode: map[string]any{"id": idC, "label": "C"}}}, http.StatusOK)
	testutil.MustReq(t, srv, "PATCH", rowPath(idC), map[string]any{"data": map[string]any{rlcolumnCode: map[string]any{"id": idA, "label": "A"}}}, http.StatusOK)

	// Must terminate, not loop forever.
	gb := testutil.MustReq(t, srv, "GET",
		fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f/graph", wsCode, tblCode, idA),
		nil, http.StatusOK)

	g := testutil.Obj(t, gb)
	nodes := testutil.Arr(t, g["nodes"])
	edges := testutil.Arr(t, g["edges"])

	// Each node appears exactly once despite the cycle.
	if len(nodes) != 3 {
		t.Errorf("cycle: expected 3 nodes, got %d", len(nodes))
	}
	// All three edges are recorded (A→B, B→C, C→A back-edge included).
	if len(edges) != 3 {
		t.Errorf("cycle: expected 3 edges, got %d", len(edges))
	}

	wantRoot := fmt.Sprintf("%s:%.0f", tableCode, idA)
	if g["root"] != wantRoot {
		t.Errorf("root: got %v, want %s", g["root"], wantRoot)
	}
}

func TestGraphOrphanedLink(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "graphorp"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	projb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "projects"}, http.StatusCreated)
	projCode := testutil.Field(t, testutil.Obj(t, projb), "code")

	taskb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "tasks"}, http.StatusCreated)
	taskCode := testutil.Field(t, testutil.Obj(t, taskb), "code")

	rlcb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, taskCode), map[string]any{
		"name": "project", "type": "row-link", "options": map[string]any{"targetTableCode": projCode},
	}, http.StatusCreated)
	rlcolumnCode := testutil.Field(t, testutil.Obj(t, rlcb), "code")

	// Create a project row then delete it immediately.
	prb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, projCode), map[string]any{"data": map[string]any{}}, http.StatusCreated)
	projRowID := testutil.Obj(t, prb)["id"].(float64)
	testutil.MustReq(t, srv, "DELETE", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, projCode, projRowID), nil, http.StatusNoContent)

	// Task linked to the now-deleted project.
	trb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, taskCode),
		map[string]any{"data": map[string]any{rlcolumnCode: map[string]any{"id": projRowID, "label": "Ghost"}}},
		http.StatusCreated)
	taskRowID := testutil.Obj(t, trb)["id"].(float64)

	gb := testutil.MustReq(t, srv, "GET",
		fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f/graph", wsCode, taskCode, taskRowID),
		nil, http.StatusOK)

	g := testutil.Obj(t, gb)
	nodes := testutil.Arr(t, g["nodes"])
	edges := testutil.Arr(t, g["edges"])

	// Task node + ghost node for deleted project.
	if len(nodes) != 2 {
		t.Errorf("orphan: expected 2 nodes, got %d", len(nodes))
	}
	if len(edges) != 1 {
		t.Errorf("orphan: expected 1 edge, got %d", len(edges))
	}

	wantGhostID := fmt.Sprintf("%s:%.0f", projCode, projRowID)
	var ghost map[string]any
	for _, n := range nodes {
		nd := testutil.Obj(t, n)
		if nd["id"] == wantGhostID {
			ghost = nd
		}
	}
	if ghost == nil {
		t.Fatal("ghost node not found in graph")
	}
	// Ghost data is null.
	if ghost["data"] != nil {
		t.Errorf("ghost node data: got %v, want nil/null", ghost["data"])
	}
	// Ghost label carries the stale label from the stored link.
	label, _ := ghost["label"].(string)
	if label == "" {
		t.Error("ghost node label should not be empty")
	}
}

func TestGraphDepthLimit(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "graphdep"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "chain"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")
	tableCode := testutil.Field(t, testutil.Obj(t, tb), "code") // used as targettableCode

	rlcb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode), map[string]any{
		"name": "next", "type": "row-link", "options": map[string]any{"targetTableCode": tableCode},
	}, http.StatusCreated)
	rlcolumnCode := testutil.Field(t, testutil.Obj(t, rlcb), "code")

	// Build a linear chain of 8 rows: 0→1→2→…→7
	const chainLen = 8
	ids := make([]float64, chainLen)
	for i := 0; i < chainLen; i++ {
		r := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode), map[string]any{"data": map[string]any{}}, http.StatusCreated)
		ids[i] = testutil.Obj(t, r)["id"].(float64)
	}
	for i := 0; i < chainLen-1; i++ {
		testutil.MustReq(t, srv, "PATCH",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, tblCode, ids[i]),
			map[string]any{"data": map[string]any{rlcolumnCode: map[string]any{"id": ids[i+1], "label": fmt.Sprintf("node%d", i+1)}}},
			http.StatusOK)
	}

	t.Run("default depth 5 yields 6 nodes (0..5)", func(t *testing.T) {
		gb := testutil.MustReq(t, srv, "GET",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f/graph", wsCode, tblCode, ids[0]),
			nil, http.StatusOK)
		nodes := testutil.Arr(t, testutil.Obj(t, gb)["nodes"])
		if len(nodes) != 6 {
			t.Errorf("depth 5: expected 6 nodes, got %d", len(nodes))
		}
	})

	t.Run("depth=2 yields 3 nodes (0..2)", func(t *testing.T) {
		gb := testutil.MustReq(t, srv, "GET",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f/graph?depth=2", wsCode, tblCode, ids[0]),
			nil, http.StatusOK)
		nodes := testutil.Arr(t, testutil.Obj(t, gb)["nodes"])
		if len(nodes) != 3 {
			t.Errorf("depth 2: expected 3 nodes, got %d", len(nodes))
		}
	})

	t.Run("depth clamped to 10 still traverses full 8-node chain", func(t *testing.T) {
		gb := testutil.MustReq(t, srv, "GET",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f/graph?depth=999", wsCode, tblCode, ids[0]),
			nil, http.StatusOK)
		nodes := testutil.Arr(t, testutil.Obj(t, gb)["nodes"])
		if len(nodes) != chainLen {
			t.Errorf("clamped depth: expected %d nodes, got %d", chainLen, len(nodes))
		}
	})

	t.Run("non-existent row returns 404", func(t *testing.T) {
		testutil.MustReq(t, srv, "GET",
			fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/99999/graph", wsCode, tblCode),
			nil, http.StatusNotFound)
	})
}
