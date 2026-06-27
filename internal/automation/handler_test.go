package automation_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/saintedlama/degubase/internal/testutil"
)

func TestScripts(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "t"}, http.StatusCreated)
	tblID := testutil.Obj(t, tb)["id"].(float64)

	base := fmt.Sprintf("/api/workspaces/%s/scripts/", wsCode)

	t.Run("list empty", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 0 {
			t.Error("expected empty script list")
		}
	})

	var scriptID float64

	t.Run("create", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name":       "on-create",
			"event_type": "record.created",
			"code":       "-- noop",
			"enabled":    true,
			"table_ids":  []float64{tblID},
		}, http.StatusCreated)
		o := testutil.Obj(t, b)
		scriptID = o["id"].(float64)
		if testutil.Field(t, o, "name") != "on-create" {
			t.Errorf("name mismatch")
		}
		if testutil.Field(t, o, "event_type") != "record.created" {
			t.Errorf("event_type mismatch")
		}
	})

	t.Run("create invalid event_type returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name": "bad", "event_type": "not.valid", "code": "",
		}, http.StatusBadRequest)
	})

	scriptPath := func() string { return fmt.Sprintf("%s%.0f", base, scriptID) }

	t.Run("get", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", scriptPath(), nil, http.StatusOK)
		o := testutil.Obj(t, b)
		if o["id"].(float64) != scriptID {
			t.Errorf("id mismatch")
		}
	})

	t.Run("list shows created script", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 1 {
			t.Errorf("expected 1 script, got %d", len(testutil.Arr(t, b)))
		}
	})

	t.Run("update", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "PUT", scriptPath(), map[string]any{
			"name":       "on-create-v2",
			"event_type": "record.created",
			"code":       "-- updated",
			"enabled":    false,
			"table_ids":  []float64{tblID},
		}, http.StatusOK)
		o := testutil.Obj(t, b)
		if testutil.Field(t, o, "name") != "on-create-v2" {
			t.Errorf("name not updated")
		}
	})

	t.Run("list executions empty", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", scriptPath()+"/executions", nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 0 {
			t.Error("expected empty executions")
		}
	})

	t.Run("create execution", func(t *testing.T) {
		now := time.Now().UTC().Format(time.RFC3339)
		b := testutil.MustReq(t, srv, "POST", scriptPath()+"/executions", map[string]any{
			"event_type":  "record.created",
			"started_at":  now,
			"ended_at":    now,
			"duration_ms": 10,
			"success":     true,
			"logs":        []any{},
		}, http.StatusCreated)
		o := testutil.Obj(t, b)
		if o["script_id"].(float64) != scriptID {
			t.Errorf("script_id mismatch")
		}
		if o["success"] != true {
			t.Errorf("success should be true")
		}
	})

	t.Run("list executions after create", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", scriptPath()+"/executions", nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 1 {
			t.Errorf("expected 1 execution, got %d", len(testutil.Arr(t, b)))
		}
	})

	t.Run("delete", func(t *testing.T) {
		testutil.MustReq(t, srv, "DELETE", scriptPath(), nil, http.StatusNoContent)
		b := testutil.MustReq(t, srv, "GET", base, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 0 {
			t.Error("expected empty list after delete")
		}
	})
}

func TestScriptValidation(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")
	base := fmt.Sprintf("/api/workspaces/%s/scripts/", wsCode)
	envBase := fmt.Sprintf("/api/workspaces/%s/scripts/env", wsCode)

	t.Run("create missing name returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST", base, map[string]any{
			"event_type": "record.created", "code": "-- x",
		}, http.StatusBadRequest)
	})

	t.Run("create missing code returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name": "x", "event_type": "record.created",
		}, http.StatusBadRequest)
	})

	t.Run("create missing event_type returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST", base, map[string]any{
			"name": "x", "code": "-- x",
		}, http.StatusBadRequest)
	})

	// Create a valid script to test update/get/delete error branches.
	b := testutil.MustReq(t, srv, "POST", base, map[string]any{
		"name": "valid", "event_type": "record.created", "code": "-- ok",
	}, http.StatusCreated)
	scriptID := testutil.Obj(t, b)["id"].(float64)
	scriptPath := fmt.Sprintf("%s%.0f", base, scriptID)

	t.Run("update with non-numeric ID returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "PUT", base+"notanid", map[string]any{
			"name": "x", "event_type": "record.created", "code": "-- x",
		}, http.StatusBadRequest)
	})

	t.Run("update missing name returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "PUT", scriptPath, map[string]any{
			"event_type": "record.created", "code": "-- x",
		}, http.StatusBadRequest)
	})

	t.Run("get non-existent script returns 404", func(t *testing.T) {
		testutil.MustReq(t, srv, "GET", base+"99999", nil, http.StatusNotFound)
	})

	t.Run("delete with non-numeric ID returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "DELETE", base+"notanid", nil, http.StatusBadRequest)
	})

	t.Run("create execution with non-numeric script ID returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST", base+"notanid/executions", map[string]any{}, http.StatusBadRequest)
	})

	t.Run("list executions with non-numeric script ID returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "GET", base+"notanid/executions", nil, http.StatusBadRequest)
	})

	t.Run("upsert env missing key returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST", envBase, map[string]any{"value": "v"}, http.StatusBadRequest)
	})

	t.Run("delete env with non-numeric ID returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "DELETE", envBase+"/notanid", nil, http.StatusBadRequest)
	})
}

func TestScriptEnv(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")
	envBase := fmt.Sprintf("/api/workspaces/%s/scripts/env", wsCode)

	t.Run("list empty", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", envBase, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 0 {
			t.Error("expected empty env list")
		}
	})

	var envID float64

	t.Run("upsert creates env var", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", envBase, map[string]any{
			"key": "API_KEY", "value": "secret123", "is_secret": false,
		}, http.StatusOK)
		o := testutil.Obj(t, b)
		envID = o["id"].(float64)
		if testutil.Field(t, o, "key") != "API_KEY" {
			t.Errorf("key mismatch")
		}
	})

	t.Run("list shows env var", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", envBase, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 1 {
			t.Errorf("expected 1 env var, got %d", len(testutil.Arr(t, b)))
		}
	})

	t.Run("upsert updates existing key", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST", envBase, map[string]any{
			"key": "API_KEY", "value": "updated", "is_secret": false,
		}, http.StatusOK)
		b := testutil.MustReq(t, srv, "GET", envBase, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 1 {
			t.Error("upsert should not create duplicate")
		}
	})

	t.Run("delete env var", func(t *testing.T) {
		testutil.MustReq(t, srv, "DELETE", fmt.Sprintf("%s/%.0f", envBase, envID), nil, http.StatusNoContent)
		b := testutil.MustReq(t, srv, "GET", envBase, nil, http.StatusOK)
		if len(testutil.Arr(t, b)) != 0 {
			t.Error("expected empty list after delete")
		}
	})
}
