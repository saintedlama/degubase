package snapshots_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saintedlama/degubase/internal/infrastructure/store"
	"github.com/saintedlama/degubase/internal/snapshots"
)

func TestCreate(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	snapshotDir := filepath.Join(dir, "snapshots")

	st, err := store.New("file:" + dbPath + "?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() { st.DB().Close() })

	snap, err := snapshots.Create(st.DB(), snapshotDir)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if snap.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if snap.SizeBytes == 0 {
		t.Fatal("expected non-zero size")
	}

	snapPath := filepath.Join(snapshotDir, snap.ID+".db")
	if _, err := os.Stat(snapPath); err != nil {
		t.Fatalf("snapshot file not found at %s: %v", snapPath, err)
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()

	// Empty dir returns empty slice, no error.
	snaps, err := snapshots.List(dir)
	if err != nil {
		t.Fatalf("List on empty dir: %v", err)
	}
	if len(snaps) != 0 {
		t.Fatalf("expected 0 snapshots, got %d", len(snaps))
	}

	// Non-existent dir also returns empty slice.
	snaps, err = snapshots.List(filepath.Join(dir, "nonexistent"))
	if err != nil {
		t.Fatalf("List on missing dir: %v", err)
	}
	if len(snaps) != 0 {
		t.Fatalf("expected 0 snapshots for missing dir, got %d", len(snaps))
	}

	// Write some fake snapshots and verify ordering (newest first).
	names := []string{"20260101-000000.db", "20260103-000000.db", "20260102-000000.db"}
	for _, n := range names {
		os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644)
	}
	snaps, err = snapshots.List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(snaps) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(snaps))
	}
	if snaps[0].ID != "20260103-000000" {
		t.Errorf("expected newest first, got %s", snaps[0].ID)
	}
	if snaps[2].ID != "20260101-000000" {
		t.Errorf("expected oldest last, got %s", snaps[2].ID)
	}
}

func TestPrune(t *testing.T) {
	dir := t.TempDir()

	// Create 5 fake snapshot files with distinct names.
	names := []string{
		"20260101-000000.db",
		"20260102-000000.db",
		"20260103-000000.db",
		"20260104-000000.db",
		"20260105-000000.db",
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatalf("create fake snapshot %s: %v", n, err)
		}
	}

	// Keep 3 — the 2 oldest should be deleted.
	if err := snapshots.Prune(dir, 3); err != nil {
		t.Fatalf("Prune: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 3 {
		t.Fatalf("expected 3 snapshots after prune, got %d", len(entries))
	}
	// Only the 3 newest should remain.
	for _, e := range entries {
		if e.Name() < "20260103-000000.db" {
			t.Errorf("old snapshot %s was not deleted", e.Name())
		}
	}
}

func TestRestoreOnStartup_NoMarker(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	// No marker — must be a no-op.
	if err := snapshots.RestoreOnStartup(dbPath, filepath.Join(dir, "snapshots")); err != nil {
		t.Fatalf("expected no error when no marker present, got: %v", err)
	}
}

func TestRestoreOnStartup_AppliesSnapshot(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	snapshotDir := filepath.Join(dir, "snapshots")

	// Create the original DB and insert a sentinel row.
	st, err := store.New("file:" + dbPath + "?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	if _, err := st.DB().Exec(`CREATE TABLE IF NOT EXISTS sentinel (v TEXT)`); err != nil {
		t.Fatalf("create sentinel table: %v", err)
	}
	if _, err := st.DB().Exec(`INSERT INTO sentinel VALUES ('original')`); err != nil {
		t.Fatalf("insert original: %v", err)
	}

	// Take a snapshot at this point (only has 'original').
	snap, err := snapshots.Create(st.DB(), snapshotDir)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Add a second row to the live DB (should disappear after restore).
	if _, err := st.DB().Exec(`INSERT INTO sentinel VALUES ('post-snapshot')`); err != nil {
		t.Fatalf("insert post-snapshot: %v", err)
	}
	st.DB().Close()

	// Simulate stale WAL/SHM files left behind by a hard shutdown before restore.
	os.WriteFile(dbPath+"-wal", []byte("stale"), 0o644)
	os.WriteFile(dbPath+"-shm", []byte("stale"), 0o644)

	// Write the restore marker and run the startup restore.
	if err := snapshots.WriteRestoreMarker(dbPath, snapshotDir, snap.ID); err != nil {
		t.Fatalf("WriteRestoreMarker: %v", err)
	}
	if err := snapshots.RestoreOnStartup(dbPath, snapshotDir); err != nil {
		t.Fatalf("RestoreOnStartup: %v", err)
	}

	// Marker must be gone.
	markerPath := filepath.Join(dir, "restore-marker")
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Fatal("marker file should have been deleted after restore")
	}

	// Pre-restore backup must exist (safety net).
	entries, _ := os.ReadDir(dir)
	var backupFound, walBackupFound, shmBackupFound bool
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "test.db.pre-restore-") {
			if strings.HasSuffix(name, "-wal") {
				walBackupFound = true
			} else if strings.HasSuffix(name, "-shm") {
				shmBackupFound = true
			} else {
				backupFound = true
			}
		}
	}
	if !backupFound {
		t.Fatal("pre-restore .db backup should exist after restore")
	}
	if !walBackupFound {
		t.Fatal("pre-restore .db-wal backup should exist after restore")
	}
	if !shmBackupFound {
		t.Fatal("pre-restore .db-shm backup should exist after restore")
	}

	// Stale WAL/SHM files must be cleaned up (prevents corruption on re-open).
	if _, err := os.Stat(dbPath + "-wal"); !os.IsNotExist(err) {
		t.Fatal("WAL file should have been deleted after restore")
	}
	if _, err := os.Stat(dbPath + "-shm"); !os.IsNotExist(err) {
		t.Fatal("SHM file should have been deleted after restore")
	}

	// Snapshot file must still exist (restore copies, not moves).
	snapPath := filepath.Join(snapshotDir, snap.ID+".db")
	if _, err := os.Stat(snapPath); err != nil {
		t.Fatalf("snapshot file should be preserved after restore: %v", err)
	}

	// Reopen the restored DB and verify only the pre-snapshot row is present.
	restored, err := store.New("file:" + dbPath + "?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("reopen restored DB: %v", err)
	}
	defer restored.DB().Close()

	rows, err := restored.DB().Query(`SELECT v FROM sentinel`)
	if err != nil {
		t.Fatalf("query sentinel: %v", err)
	}
	defer rows.Close()

	var vals []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		vals = append(vals, v)
	}

	if len(vals) != 1 || vals[0] != "original" {
		t.Fatalf("expected only 'original' after restore, got: %v", vals)
	}
}
