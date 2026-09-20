package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/saintedlama/degubase/internal/api"
	"github.com/saintedlama/degubase/internal/automation"
	"github.com/saintedlama/degubase/internal/infrastructure/storage"
	"github.com/saintedlama/degubase/internal/infrastructure/store"
	"github.com/saintedlama/degubase/internal/records"
)

var defaultUploadCfg = records.UploadConfig{
	ImageMaxBytes:     10 << 20,
	FileMaxBytes:      10 << 20,
	AllowedImageMIMEs: []string{"image/jpeg", "image/png", "image/webp", "image/gif"},
	ThumbnailWidth:    256,
}

func NewServer(t *testing.T) *httptest.Server {
	t.Helper()
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	srv := httptest.NewServer(api.NewRouter(st.DB(), nil, defaultUploadCfg, "", nil, true, nil, nil, automation.HTTPPolicy{}))
	t.Cleanup(srv.Close)
	return srv
}

func NewServerAuth(t *testing.T) *httptest.Server {
	t.Helper()
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	jwtSecret := []byte("test-secret-for-integration-tests-32b!")
	srv := httptest.NewServer(api.NewRouter(st.DB(), nil, defaultUploadCfg, "", jwtSecret, false, nil, nil, automation.HTTPPolicy{}))
	t.Cleanup(srv.Close)
	return srv
}

func NewServerWithStorage(t *testing.T) (*httptest.Server, *storage.Store) {
	t.Helper()
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	dir, err := os.MkdirTemp("", "degubase-test-files-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	fs, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatalf("storage.NewLocal: %v", err)
	}
	srv := httptest.NewServer(api.NewRouter(st.DB(), fs, defaultUploadCfg, "", nil, true, nil, nil, automation.HTTPPolicy{}))
	t.Cleanup(srv.Close)
	return srv, fs
}

func Req(t *testing.T, srv *httptest.Server, method, path string, body any) (int, any) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	hreq, err := http.NewRequestWithContext(context.Background(), method, srv.URL+path, r)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		hreq.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(hreq)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return resp.StatusCode, nil
	}
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	ct := resp.Header.Get("Content-Type")
	if !bytes.Contains([]byte(ct), []byte("json")) {
		return resp.StatusCode, string(rawBody)
	}
	var out any
	if err := json.Unmarshal(rawBody, &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp.StatusCode, out
}

func ReqCookies(t *testing.T, srv *httptest.Server, method, path string, body any, cookies []*http.Cookie) (int, any, []*http.Cookie) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	hreq, err := http.NewRequestWithContext(context.Background(), method, srv.URL+path, r)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		hreq.Header.Set("Content-Type", "application/json")
	}
	for _, c := range cookies {
		hreq.AddCookie(c)
	}
	resp, err := http.DefaultClient.Do(hreq)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return resp.StatusCode, nil, resp.Cookies()
	}
	var out any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp.StatusCode, out, resp.Cookies()
}

func MustReq(t *testing.T, srv *httptest.Server, method, path string, body any, wantStatus int) any {
	t.Helper()
	code, b := Req(t, srv, method, path, body)
	if code != wantStatus {
		t.Fatalf("%s %s: want %d, got %d — body: %v", method, path, wantStatus, code, b)
	}
	return b
}

func Obj(t *testing.T, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("expected JSON object, got %T: %v", v, v)
	}
	return m
}

func Arr(t *testing.T, v any) []any {
	t.Helper()
	a, ok := v.([]any)
	if !ok {
		t.Fatalf("expected JSON array, got %T: %v", v, v)
	}
	return a
}

func Field(t *testing.T, v map[string]any, key string) string {
	t.Helper()
	s, ok := v[key].(string)
	if !ok {
		t.Fatalf("field %q missing or not a string: %v", key, v)
	}
	return s
}
