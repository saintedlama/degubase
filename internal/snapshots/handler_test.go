package snapshots_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/saintedlama/degubase/internal/infrastructure/store"
	"github.com/saintedlama/degubase/internal/snapshots"
)

func newTestHandler(t *testing.T) (*snapshots.Handler, string, string, *atomic.Bool) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	snapshotDir := filepath.Join(dir, "snapshots")

	st, err := store.New("file:" + dbPath + "?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	// Always close the DB on test cleanup (sql.DB.Close is idempotent).
	t.Cleanup(func() { st.DB().Close() })

	var shutdownCalled atomic.Bool
	h := &snapshots.Handler{
		DB:          st.DB(),
		DBPath:      dbPath,
		SnapshotDir: snapshotDir,
		Shutdown: func() {
			// Simulate process exit: close the DB before the test calls RestoreOnStartup.
			st.DB().Close()
			shutdownCalled.Store(true)
		},
	}
	return h, dbPath, snapshotDir, &shutdownCalled
}

func newTestServer(h *snapshots.Handler) *httptest.Server {
	r := chi.NewRouter()
	r.Post("/api/admin/snapshots", h.CreateSnapshot)
	r.Post("/api/admin/snapshots/{id}/restore", h.ScheduleRestore)
	return httptest.NewServer(r)
}

func TestCreateSnapshotHandler(t *testing.T) {
	h, _, snapshotDir, _ := newTestHandler(t)
	srv := newTestServer(h)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/admin/snapshots", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /snapshots: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("want 201, got %d", resp.StatusCode)
	}

	var snap snapshots.Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if snap.ID == "" {
		t.Fatal("expected non-empty ID in response")
	}
	if snap.SizeBytes == 0 {
		t.Fatal("expected non-zero size in response")
	}

	if _, err := os.Stat(filepath.Join(snapshotDir, snap.ID+".db")); err != nil {
		t.Fatalf("snapshot file not found on disk: %v", err)
	}
}

func TestScheduleRestoreHandler(t *testing.T) {
	h, dbPath, snapshotDir, shutdownCalled := newTestHandler(t)
	srv := newTestServer(h)
	defer srv.Close()

	// Create a snapshot.
	resp, err := http.Post(srv.URL+"/api/admin/snapshots", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /snapshots: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create snapshot: want 201, got %d", resp.StatusCode)
	}

	var snap snapshots.Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}

	// Schedule the restore.
	resp2, err := http.Post(srv.URL+"/api/admin/snapshots/"+snap.ID+"/restore", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /restore: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusAccepted {
		t.Fatalf("want 202, got %d", resp2.StatusCode)
	}

	// Marker file must exist next to the DB.
	markerPath := filepath.Join(filepath.Dir(dbPath), "restore-marker")
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("restore marker not written: %v", err)
	}

	// Shutdown goroutine fires after 100 ms; wait a bit longer.
	time.Sleep(300 * time.Millisecond)
	if !shutdownCalled.Load() {
		t.Fatal("shutdown function was not called after restore")
	}

	// Confirm RestoreOnStartup can apply the marker successfully.
	if err := snapshots.RestoreOnStartup(dbPath, snapshotDir); err != nil {
		t.Fatalf("RestoreOnStartup: %v", err)
	}
}

func TestScheduleRestoreHandler_UnknownID(t *testing.T) {
	h, _, _, _ := newTestHandler(t)
	srv := newTestServer(h)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/admin/snapshots/nonexistent/restore", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /restore: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("want 500 for unknown snapshot, got %d", resp.StatusCode)
	}
}
