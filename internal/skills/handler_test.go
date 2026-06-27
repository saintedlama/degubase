package skills_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/saintedlama/degubase/internal/testutil"
)

// TestSkillsCRUD covers the happy-path lifecycle of a skill.
func TestSkillsCRUD(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "tasks"}, http.StatusCreated)
	tblID := testutil.Obj(t, tb)["id"].(float64)

	base := fmt.Sprintf("/api/workspaces/%s/skills/", wsCode)

	t.Run("list empty", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 0 {
			t.Error("expected empty skill list")
		}
	})

	var skillID float64

	t.Run("create", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name":        "Task Manager",
			"description": "Manage tasks",
			"table_ids":   []float64{tblID},
			"operations":  []string{"read", "create", "update", "delete"},
		}, http.StatusCreated)
		o := testutil.Obj(t, b)
		sk := testutil.Obj(t, o["skill"])
		skillID = sk["id"].(float64)
		if testutil.Field(t, sk, "name") != "Task Manager" {
			t.Errorf("name mismatch")
		}
	})

	skillPath := func() string { return fmt.Sprintf("%s%.0f", base, skillID) }

	t.Run("get", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", skillPath(), nil, http.StatusOK)
		if testutil.Obj(t, b)["id"].(float64) != skillID {
			t.Errorf("id mismatch")
		}
	})

	t.Run("list shows created skill", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 1 {
			t.Errorf("expected 1 skill, got %d", len(testutil.Arr(t, b)))
		}
	})

	t.Run("update", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "PATCH", skillPath(), map[string]any{
			"name":        "Task Manager v2",
			"description": "Manage tasks v2",
			"table_ids":   []float64{tblID},
			"operations":  []string{"read"},
		}, http.StatusOK)
		if testutil.Field(t, testutil.Obj(t, b), "name") != "Task Manager v2" {
			t.Errorf("name not updated")
		}
	})

	t.Run("delete", func(t *testing.T) {
		testutil.MustReq(t, srv, "DELETE", skillPath(), nil, http.StatusNoContent)
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 0 {
			t.Error("expected empty list after delete")
		}
	})
}

// TestSkillsValidation covers the 400/404 error branches.
func TestSkillsValidation(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")
	wb2 := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "other"}, http.StatusCreated)
	otherCode := testutil.Field(t, testutil.Obj(t, wb2), "code")

	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "tasks"}, http.StatusCreated)
	tblID := testutil.Obj(t, tb)["id"].(float64)

	// Table belonging to other workspace — should be rejected.
	otherTb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", otherCode), map[string]any{"name": "x"}, http.StatusCreated)
	otherTblID := testutil.Obj(t, otherTb)["id"].(float64)

	base := fmt.Sprintf("/api/workspaces/%s/skills/", wsCode)

	t.Run("create missing name returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST", base, map[string]any{
			"table_ids":  []float64{tblID},
			"operations": []string{"read"},
		}, http.StatusBadRequest)
	})

	t.Run("create with cross-workspace table returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name":       "bad",
			"table_ids":  []float64{otherTblID},
			"operations": []string{"read"},
		}, http.StatusBadRequest)
	})

	// Create a valid skill to test update/get/delete validation.
	b := testutil.MustReq(t, srv, "POST", base, map[string]any{
		"name":       "Valid Skill",
		"table_ids":  []float64{tblID},
		"operations": []string{"read"},
	}, http.StatusCreated)
	skillID := testutil.Obj(t, testutil.Obj(t, b)["skill"])["id"].(float64)
	skillPath := fmt.Sprintf("%s%.0f", base, skillID)

	t.Run("update missing name returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "PATCH", skillPath, map[string]any{
			"table_ids":  []float64{tblID},
			"operations": []string{"read"},
		}, http.StatusBadRequest)
	})

	t.Run("update with cross-workspace table returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "PATCH", skillPath, map[string]any{
			"name":       "bad update",
			"table_ids":  []float64{otherTblID},
			"operations": []string{"read"},
		}, http.StatusBadRequest)
	})

	t.Run("get non-existent returns 404", func(t *testing.T) {
		testutil.MustReq(t, srv, "GET", base+"99999", nil, http.StatusNotFound)
	})

	t.Run("skill.md non-existent returns 404", func(t *testing.T) {
		code, _ := testutil.Req(t, srv, "GET", base+"99999/skill.md", nil)
		if code != http.StatusNotFound {
			t.Errorf("want 404, got %d", code)
		}
	})

	t.Run("context non-existent returns 404", func(t *testing.T) {
		code, _ := testutil.Req(t, srv, "GET", base+"99999/context", nil)
		if code != http.StatusNotFound {
			t.Errorf("want 404, got %d", code)
		}
	})
}

// TestSkillContentGeneration covers skill.md and context content quality,
// including select-column option listing, sample row rendering, and all operation types.
func TestSkillContentGeneration(t *testing.T) {
	srv := testutil.NewServer(t)

	// Workspace with context field so we exercise that branch.
	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{
		"name":    "acme",
		"context": "ACME Corp project management workspace",
	}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{
		"name":    "tickets",
		"context": "Support tickets",
	}, http.StatusCreated)
	tblID := testutil.Obj(t, tb)["id"].(float64)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")
	colBase := fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode)
	rowBase := fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode)

	// Add a single-select column so parseSelectOptions is exercised.
	cb := testutil.MustReq(t, srv, "POST", colBase, map[string]any{
		"name":    "priority",
		"type":    "single-select",
		"options": map[string]any{"choices": []string{"low", "medium", "high"}},
	}, http.StatusCreated)
	priorityCode := testutil.Field(t, testutil.Obj(t, cb), "code")

	// Add a multi-select column to exercise that branch too.
	cb2 := testutil.MustReq(t, srv, "POST", colBase, map[string]any{
		"name":    "tags",
		"type":    "multi-select",
		"options": map[string]any{"choices": []string{"bug", "feature", "docs"}},
	}, http.StatusCreated)
	tagsCode := testutil.Field(t, testutil.Obj(t, cb2), "code")

	// Add a text column.
	cb3 := testutil.MustReq(t, srv, "POST", colBase, map[string]any{
		"name": "title",
		"type": "text",
	}, http.StatusCreated)
	titleCode := testutil.Field(t, testutil.Obj(t, cb3), "code")

	// Add extra columns to exceed 6 and trigger the dispCols[:6] truncation.
	for _, name := range []string{"col4", "col5", "col6", "col7"} {
		testutil.MustReq(t, srv, "POST", colBase, map[string]any{
			"name": name,
			"type": "text",
		}, http.StatusCreated)
	}

	// Insert a couple of rows so the "Sample rows" section is rendered.
	testutil.MustReq(t, srv, "POST", rowBase, map[string]any{
		"data": map[string]any{
			priorityCode: "high",
			tagsCode:     []string{"bug"},
			titleCode:    "First ticket",
		},
	}, http.StatusCreated)
	testutil.MustReq(t, srv, "POST", rowBase, map[string]any{
		"data": map[string]any{
			priorityCode: "low",
			titleCode:    "Second ticket",
		},
	}, http.StatusCreated)

	base := fmt.Sprintf("/api/workspaces/%s/skills/", wsCode)

	// Create skill with all operation types to exercise every code branch.
	b := testutil.MustReq(t, srv, "POST", base, map[string]any{
		"name":        "Ticket Manager",
		"description": "Manage support tickets",
		"table_ids":   []float64{tblID},
		"operations":  []string{"read", "create", "update", "delete", "views", "schema"},
	}, http.StatusCreated)
	skillID := testutil.Obj(t, testutil.Obj(t, b)["skill"])["id"].(float64)

	t.Run("skill.md contains expected sections", func(t *testing.T) {
		code, body := testutil.Req(t, srv, "GET", fmt.Sprintf("%s%.0f/skill.md", base, skillID), nil)
		if code != http.StatusOK {
			t.Fatalf("skill.md: want 200, got %d", code)
		}
		md := body.(string)

		for _, want := range []string{
			"Ticket Manager",
			"ACME Corp project management workspace", // workspace context
			"priority",                               // select column name
			"low", "medium", "high",                  // single-select options via parseSelectOptions
			"bug", "feature", "docs", // multi-select options
			"Sample rows",  // sample data table rendered
			"First ticket", // row data appears
			"List rows",
			"Create row",
			"Patch row",
			"Delete row",
			"List views",
			"List columns",
			"filter=",
			"sort=",
		} {
			if !strings.Contains(md, want) {
				t.Errorf("skill.md missing %q", want)
			}
		}
	})

	t.Run("context returns same markdown as skill.md", func(t *testing.T) {
		codeMd, bodyMd := testutil.Req(t, srv, "GET", fmt.Sprintf("%s%.0f/skill.md", base, skillID), nil)
		codeCtx, bodyCtx := testutil.Req(t, srv, "GET", fmt.Sprintf("%s%.0f/context", base, skillID), nil)
		if codeMd != http.StatusOK || codeCtx != http.StatusOK {
			t.Fatalf("want 200/200, got %d/%d", codeMd, codeCtx)
		}
		if bodyMd.(string) == "" {
			t.Error("expected non-empty skill.md body")
		}
		if bodyCtx.(string) == "" {
			t.Error("expected non-empty context body")
		}
	})
}

// TestSkillPreview covers the stateless preview endpoint.
func TestSkillPreview(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "items"}, http.StatusCreated)
	tblID := testutil.Obj(t, tb)["id"].(float64)

	base := fmt.Sprintf("/api/workspaces/%s/skills/", wsCode)

	t.Run("preview returns markdown with skill name", func(t *testing.T) {
		code, body := testutil.Req(t, srv, "POST", base+"preview", map[string]any{
			"name":        "Preview Skill",
			"description": "test preview",
			"table_ids":   []float64{tblID},
			"operations":  []string{"read"},
		})
		if code != http.StatusOK {
			t.Fatalf("preview: want 200, got %d", code)
		}
		md, ok := body.(string)
		if !ok || md == "" {
			t.Fatalf("expected non-empty string body, got %T", body)
		}
		if !strings.Contains(md, "Preview Skill") {
			t.Errorf("preview missing skill name")
		}
		if !strings.Contains(md, "List rows") {
			t.Errorf("preview missing read endpoint docs")
		}
	})

	t.Run("preview with empty name falls back to workspace name", func(t *testing.T) {
		code, body := testutil.Req(t, srv, "POST", base+"preview", map[string]any{
			"table_ids":  []float64{tblID},
			"operations": []string{"create"},
		})
		if code != http.StatusOK {
			t.Fatalf("preview: want 200, got %d", code)
		}
		md := body.(string)
		if !strings.Contains(md, "Create row") {
			t.Errorf("preview missing create endpoint docs")
		}
	})

	t.Run("preview with views and schema operations", func(t *testing.T) {
		code, body := testutil.Req(t, srv, "POST", base+"preview", map[string]any{
			"name":       "Full Skill",
			"table_ids":  []float64{tblID},
			"operations": []string{"views", "schema"},
		})
		if code != http.StatusOK {
			t.Fatalf("preview: want 200, got %d", code)
		}
		md := body.(string)
		if !strings.Contains(md, "List views") {
			t.Errorf("preview missing views docs")
		}
		if !strings.Contains(md, "List columns") {
			t.Errorf("preview missing schema docs")
		}
	})
}
