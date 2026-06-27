package identity_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/saintedlama/degubase/internal/testutil"
)

// ── Auth disabled ──────────────────────────────────────────────────────────────

func TestAuthDisabled(t *testing.T) {
	srv := testutil.NewServer(t)

	t.Run("setup returns auth_disabled", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", "/api/auth/setup", nil, http.StatusOK)
		o := testutil.Obj(t, b)
		if s, ok := o["setup_required"].(bool); !ok || s != false {
			t.Errorf("setup_required want false, got %v", o["setup_required"])
		}
		if s, ok := o["auth_disabled"].(bool); !ok || s != true {
			t.Errorf("auth_disabled want true, got %v", o["auth_disabled"])
		}
	})

	t.Run("me returns default user", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", "/api/auth/me", nil, http.StatusOK)
		o := testutil.Obj(t, b)
		if testutil.Field(t, o, "username") != "system" {
			t.Errorf("username want system, got %s", testutil.Field(t, o, "username"))
		}
		if testutil.Field(t, o, "name") != "System" {
			t.Errorf("name want System, got %s", testutil.Field(t, o, "name"))
		}
		if o["id"] == nil {
			t.Errorf("id missing")
		}
	})

	t.Run("login returns 404", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", "/api/auth/login", map[string]string{
			"username": "test",
			"password": "test",
		}, http.StatusNotFound)
		o := testutil.Obj(t, b)
		if testutil.Field(t, o, "code") != "auth_disabled" {
			t.Errorf("code want auth_disabled, got %s", testutil.Field(t, o, "code"))
		}
	})

	t.Run("logout returns 404", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", "/api/auth/logout", nil, http.StatusNotFound)
		o := testutil.Obj(t, b)
		if testutil.Field(t, o, "code") != "auth_disabled" {
			t.Errorf("code want auth_disabled, got %s", testutil.Field(t, o, "code"))
		}
	})

	t.Run("user management returns 404 when auth is disabled", func(t *testing.T) {
		testutil.MustReq(t, srv, "GET", "/api/admin/users", nil, http.StatusNotFound)
		testutil.MustReq(t, srv, "POST", "/api/admin/users", map[string]string{
			"username": "x",
			"password": "x",
		}, http.StatusNotFound)
	})

	t.Run("token routes return 404", func(t *testing.T) {
		wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
		wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

		testutil.MustReq(t, srv, "GET", fmt.Sprintf("/api/workspaces/%s/tokens", wsCode), nil, http.StatusNotFound)
		testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tokens", wsCode), map[string]string{
			"name": "my-token",
		}, http.StatusNotFound)
	})

	t.Run("workspace crud works without auth", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{
			"name":    "no-auth-ws",
			"context": "created without auth",
		}, http.StatusCreated)
		o := testutil.Obj(t, b)
		if testutil.Field(t, o, "name") != "no-auth-ws" {
			t.Errorf("name mismatch")
		}
		wsCode := testutil.Field(t, o, "code")

		b = testutil.MustReq(t, srv, "GET", fmt.Sprintf("/api/workspaces/%s/", wsCode), nil, http.StatusOK)
		o = testutil.Obj(t, b)
		if testutil.Field(t, o, "name") != "no-auth-ws" {
			t.Errorf("get name mismatch")
		}

		testutil.MustReq(t, srv, "DELETE", fmt.Sprintf("/api/workspaces/%s/", wsCode), nil, http.StatusNoContent)
	})

	t.Run("table crud works without auth", func(t *testing.T) {
		wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
		wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

		tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{
			"name": "no-auth-table",
		}, http.StatusCreated)
		tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")

		tb = testutil.MustReq(t, srv, "GET", fmt.Sprintf("/api/workspaces/%s/tables/%s/", wsCode, tblCode), nil, http.StatusOK)
		if testutil.Field(t, testutil.Obj(t, tb), "name") != "no-auth-table" {
			t.Errorf("get name mismatch")
		}
	})

	t.Run("row crud works without auth", func(t *testing.T) {
		wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
		wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

		tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "t"}, http.StatusCreated)
		tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")

		rb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode), map[string]any{
			"data": map[string]any{"title": "no auth row"},
		}, http.StatusCreated)
		rowID := testutil.Obj(t, rb)["id"].(float64)

		testutil.MustReq(t, srv, "PUT", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, tblCode, rowID), map[string]any{
			"data": map[string]any{"title": "updated"},
		}, http.StatusOK)

		testutil.MustReq(t, srv, "DELETE", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/%.0f", wsCode, tblCode, rowID), nil, http.StatusNoContent)
	})

	t.Run("default user is consistent across requests", func(t *testing.T) {
		b1 := testutil.MustReq(t, srv, "GET", "/api/auth/me", nil, http.StatusOK)
		b2 := testutil.MustReq(t, srv, "GET", "/api/auth/me", nil, http.StatusOK)
		o1 := testutil.Obj(t, b1)
		o2 := testutil.Obj(t, b2)
		if o1["id"] != o2["id"] {
			t.Errorf("default user id should be consistent, got %v and %v", o1["id"], o2["id"])
		}
		if testutil.Field(t, o1, "username") != testutil.Field(t, o2, "username") {
			t.Errorf("default user should be consistent")
		}
	})
}

// ── Auth validation ────────────────────────────────────────────────────────────

func TestAuthValidation(t *testing.T) {
	srv := testutil.NewServerAuth(t)

	t.Run("login missing credentials returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "POST", "/api/auth/login", map[string]string{
			"username": "admin",
		}, http.StatusBadRequest)
	})

	t.Run("login wrong password returns 401", func(t *testing.T) {
		// Bootstrap the first user.
		testutil.MustReq(t, srv, "POST", "/api/auth/login", map[string]string{
			"username": "admin", "password": "correct",
		}, http.StatusOK)

		code, _ := testutil.Req(t, srv, "POST", "/api/auth/login", map[string]string{
			"username": "admin", "password": "wrong",
		})
		if code != http.StatusUnauthorized {
			t.Errorf("wrong password: want 401, got %d", code)
		}
	})

	t.Run("login unknown user returns 401", func(t *testing.T) {
		code, _ := testutil.Req(t, srv, "POST", "/api/auth/login", map[string]string{
			"username": "ghost", "password": "anything",
		})
		if code != http.StatusUnauthorized {
			t.Errorf("unknown user: want 401, got %d", code)
		}
	})
}

// ── Auth enabled ───────────────────────────────────────────────────────────────

func TestAuthEnabled(t *testing.T) {
	srv := testutil.NewServerAuth(t)

	t.Run("setup reports no users", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", "/api/auth/setup", nil, http.StatusOK)
		o := testutil.Obj(t, b)
		if s, ok := o["setup_required"].(bool); !ok || s != true {
			t.Errorf("setup_required want true, got %v", o["setup_required"])
		}
		if _, ok := o["auth_disabled"]; ok {
			t.Errorf("auth_disabled should not be present when auth is enabled")
		}
	})

	t.Run("protected routes require auth", func(t *testing.T) {
		testutil.MustReq(t, srv, "GET", "/api/workspaces/", nil, http.StatusUnauthorized)
		testutil.MustReq(t, srv, "GET", "/api/auth/me", nil, http.StatusUnauthorized)
		testutil.MustReq(t, srv, "GET", "/api/admin/users", nil, http.StatusUnauthorized)
	})

	var cookies []*http.Cookie
	var user2ID float64

	t.Run("login bootstraps first admin", func(t *testing.T) {
		code, body, respCookies := testutil.ReqCookies(t, srv, "POST", "/api/auth/login", map[string]string{
			"username": "admin",
			"password": "admin123",
		}, nil)

		if code != http.StatusOK {
			t.Fatalf("login: want 200, got %d — body: %v", code, body)
		}
		o := testutil.Obj(t, body)
		if testutil.Field(t, o, "username") != "admin" {
			t.Errorf("username want admin, got %s", testutil.Field(t, o, "username"))
		}
		if o["is_admin"] != true {
			t.Errorf("first user should be admin")
		}
		cookies = respCookies
	})

	t.Run("me returns authenticated user", func(t *testing.T) {
		code, body, _ := testutil.ReqCookies(t, srv, "GET", "/api/auth/me", nil, cookies)
		if code != http.StatusOK {
			t.Fatalf("me: want 200, got %d", code)
		}
		o := testutil.Obj(t, body)
		if testutil.Field(t, o, "username") != "admin" {
			t.Errorf("username want admin, got %s", testutil.Field(t, o, "username"))
		}
	})

	t.Run("admin list users", func(t *testing.T) {
		code, body, _ := testutil.ReqCookies(t, srv, "GET", "/api/admin/users", nil, cookies)
		if code != http.StatusOK {
			t.Fatalf("list users: want 200, got %d", code)
		}
		users := testutil.Arr(t, body)
		if len(users) != 1 {
			t.Errorf("want 1 user, got %d", len(users))
		}
	})

	t.Run("admin create user", func(t *testing.T) {
		code, body, _ := testutil.ReqCookies(t, srv, "POST", "/api/admin/users", map[string]string{
			"username": "user2",
			"name":     "User Two",
			"password": "pass123",
		}, cookies)
		if code != http.StatusCreated {
			t.Fatalf("create user: want 201, got %d — body: %v", code, body)
		}
		o := testutil.Obj(t, body)
		if testutil.Field(t, o, "username") != "user2" {
			t.Errorf("username want user2, got %s", testutil.Field(t, o, "username"))
		}
		user2ID = o["id"].(float64)
	})

	t.Run("login as second user", func(t *testing.T) {
		code, body, respCookies := testutil.ReqCookies(t, srv, "POST", "/api/auth/login", map[string]string{
			"username": "user2",
			"password": "pass123",
		}, nil)
		if code != http.StatusOK {
			t.Fatalf("login user2: want 200, got %d — body: %v", code, body)
		}
		o := testutil.Obj(t, body)
		if testutil.Field(t, o, "username") != "user2" {
			t.Errorf("username want user2, got %s", testutil.Field(t, o, "username"))
		}
		if o["is_admin"] != false {
			t.Errorf("user2 should not be admin")
		}

		code, _, _ = testutil.ReqCookies(t, srv, "GET", "/api/admin/users", nil, respCookies)
		if code != http.StatusForbidden {
			t.Errorf("non-admin accessing /api/admin/users: want 403, got %d", code)
		}
	})

	t.Run("admin delete user", func(t *testing.T) {
		code, body, _ := testutil.ReqCookies(t, srv, "DELETE", fmt.Sprintf("/api/admin/users/%.0f", user2ID), nil, cookies)
		if code != http.StatusNoContent {
			t.Fatalf("delete user: want 204, got %d — body: %v", code, body)
		}
		code, body, _ = testutil.ReqCookies(t, srv, "GET", "/api/admin/users", nil, cookies)
		if code != http.StatusOK {
			t.Fatalf("list users: want 200, got %d", code)
		}
		if len(testutil.Arr(t, body)) != 1 {
			t.Errorf("want 1 user after delete, got %d", len(testutil.Arr(t, body)))
		}
	})

	t.Run("protected routes work with auth", func(t *testing.T) {
		code, body, _ := testutil.ReqCookies(t, srv, "GET", "/api/workspaces/", nil, cookies)
		if code != http.StatusOK {
			t.Fatalf("list workspaces: want 200, got %d — body: %v", code, body)
		}
	})

	t.Run("logout clears session", func(t *testing.T) {
		code, _, respCookies := testutil.ReqCookies(t, srv, "POST", "/api/auth/logout", nil, cookies)
		if code != http.StatusNoContent {
			t.Fatalf("logout: want 204, got %d", code)
		}
		if len(respCookies) == 0 {
			t.Errorf("logout should return cleared cookie")
		}
	})

	t.Run("workspace tokens in auth mode", func(t *testing.T) {
		code, body, _ := testutil.ReqCookies(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, cookies)
		if code != http.StatusCreated {
			t.Fatalf("create workspace: want 201, got %d — body: %v", code, body)
		}
		wsCode := testutil.Field(t, testutil.Obj(t, body), "code")

		code, body, _ = testutil.ReqCookies(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tokens", wsCode), map[string]string{
			"name": "agent-token",
		}, cookies)
		if code != http.StatusCreated {
			t.Fatalf("create token: want 201, got %d — body: %v", code, body)
		}
		o := testutil.Obj(t, body)
		tokenValue := testutil.Field(t, o, "token")
		if !strings.HasPrefix(tokenValue, "degu_") {
			t.Errorf("token should start with degu_, got %s", tokenValue)
		}
		tokenID := o["id"].(float64)

		code, body, _ = testutil.ReqCookies(t, srv, "GET", fmt.Sprintf("/api/workspaces/%s/tokens", wsCode), nil, cookies)
		if code != http.StatusOK {
			t.Fatalf("list tokens: want 200, got %d", code)
		}
		if len(testutil.Arr(t, body)) != 1 {
			t.Errorf("want 1 token, got %d", len(testutil.Arr(t, body)))
		}

		code, body, _ = testutil.ReqCookies(t, srv, "DELETE", fmt.Sprintf("/api/workspaces/%s/tokens/%.0f", wsCode, tokenID), nil, cookies)
		if code != http.StatusNoContent {
			t.Fatalf("delete token: want 204, got %d — body: %v", code, body)
		}
		code, body, _ = testutil.ReqCookies(t, srv, "GET", fmt.Sprintf("/api/workspaces/%s/tokens", wsCode), nil, cookies)
		if code != http.StatusOK {
			t.Fatalf("list tokens after delete: want 200, got %d", code)
		}
		if len(testutil.Arr(t, body)) != 0 {
			t.Errorf("want 0 tokens after delete, got %d", len(testutil.Arr(t, body)))
		}
	})
}
