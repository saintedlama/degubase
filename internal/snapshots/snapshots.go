package snapshots

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const markerFile = "restore-marker"

// Snapshot describes a database snapshot file.
type Snapshot struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	SizeBytes int64     `json:"size_bytes"`
}

// Create takes a live snapshot of the database via VACUUM INTO and writes it to
// snapshotDir. The returned Snapshot.ID is the bare filename stem (no extension).
func Create(db *sql.DB, snapshotDir string) (*Snapshot, error) {
	if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
		return nil, fmt.Errorf("create snapshot dir: %w", err)
	}

	id := time.Now().UTC().Format("20060102-150405")
	destPath := filepath.Join(snapshotDir, id+".db")

	slog.Info("creating snapshot", "id", id, "dest", destPath)

	// VACUUM INTO creates a compacted, consistent copy of the live database.
	// The path is constructed from a timestamp so it is safe to interpolate.
	if _, err := db.Exec(fmt.Sprintf("VACUUM INTO '%s'", destPath)); err != nil {
		slog.Error("snapshot vacuum failed", "id", id, "error", err)
		return nil, fmt.Errorf("vacuum into: %w", err)
	}

	info, err := os.Stat(destPath)
	if err != nil {
		return nil, fmt.Errorf("stat snapshot: %w", err)
	}

	slog.Info("snapshot created", "id", id, "size_bytes", info.Size())
	return &Snapshot{
		ID:        id,
		CreatedAt: info.ModTime().UTC(),
		SizeBytes: info.Size(),
	}, nil
}

// List returns all snapshots in snapshotDir, newest first.
func List(snapshotDir string) ([]Snapshot, error) {
	entries, err := os.ReadDir(snapshotDir)
	if os.IsNotExist(err) {
		return []Snapshot{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read snapshot dir: %w", err)
	}

	var result []Snapshot
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".db")
		t, _ := time.Parse("20060102-150405", id)
		result = append(result, Snapshot{
			ID:        id,
			CreatedAt: t.UTC(),
			SizeBytes: info.Size(),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID > result[j].ID })
	return result, nil
}

// RestoreOnStartup must be called before the database is opened. It checks for a
// pending restore marker and, if present, atomically replaces the database file
// with the chosen snapshot before returning.
func RestoreOnStartup(dbPath, snapshotDir string) error {
	markerPath := filepath.Join(filepath.Dir(dbPath), markerFile)
	data, err := os.ReadFile(markerPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read restore marker: %w", err)
	}

	snapshotName := strings.TrimSpace(string(data))
	slog.Info("restore marker found", "marker", markerPath, "snapshot", snapshotName)

	snapshotPath := filepath.Join(snapshotDir, snapshotName)

	info, err := os.Stat(snapshotPath)
	if err != nil {
		slog.Error("snapshot file missing for restore", "path", snapshotPath, "error", err)
		return fmt.Errorf("snapshot file missing (%s): %w", snapshotPath, err)
	}
	slog.Info("restoring from snapshot", "snapshot", snapshotName, "size_bytes", info.Size())

	// Safety net: back up the current database and its journal files before
	// replacing them, so the operator can manually roll back if the restored
	// snapshot is broken.
	backupPath := dbPath + ".pre-restore-" + time.Now().UTC().Format("20060102-150405")
	if _, err := os.Stat(dbPath); err == nil {
		slog.Info("backing up current database before restore", "src", dbPath, "dst", backupPath)
		if err := copyFile(dbPath, backupPath); err != nil {
			slog.Error("pre-restore backup failed", "src", dbPath, "dst", backupPath, "error", err)
			return fmt.Errorf("pre-restore backup: %w", err)
		}
		slog.Info("pre-restore backup saved", "path", filepath.Base(backupPath))
	}
	for _, ext := range []string{"-wal", "-shm"} {
		src := dbPath + ext
		dst := backupPath + ext
		if _, err := os.Stat(src); err == nil {
			slog.Info("backing up journal file before restore", "src", filepath.Base(src), "dst", filepath.Base(dst))
			if err := copyFile(src, dst); err != nil {
				slog.Error("pre-restore journal backup failed", "src", src, "dst", dst, "error", err)
				return fmt.Errorf("pre-restore journal backup: %w", err)
			}
			slog.Info("pre-restore journal backup saved", "path", filepath.Base(dst))
		}
	}

	// Copy snapshot to a temp file in the same directory as the DB so that the
	// subsequent rename is atomic (same filesystem, same directory).
	tmpPath := dbPath + ".restore-tmp"
	slog.Info("copying snapshot to temp file", "src", snapshotPath, "dst", tmpPath)
	if err := copyFile(snapshotPath, tmpPath); err != nil {
		slog.Error("copy snapshot failed", "src", snapshotPath, "dst", tmpPath, "error", err)
		slog.Error("Pre-restore backup is safe at " + backupPath + ", database unchanged — delete the restore marker and restart")
		return fmt.Errorf("copy snapshot: %w", err)
	}
	slog.Info("snapshot copied to temp file", "path", tmpPath)

	slog.Info("replacing database file", "tmp", tmpPath, "db", dbPath)
	if err := os.Rename(tmpPath, dbPath); err != nil {
		os.Remove(tmpPath)
		slog.Error("replace database failed", "tmp", tmpPath, "db", dbPath, "error", err)
		slog.Error("Pre-restore backup is safe at " + backupPath + " — to roll back: cp " + backupPath + "* " + dbPath + "* && restart")
		return fmt.Errorf("replace database: %w", err)
	}
	slog.Info("database replaced with snapshot")

	// Clean up WAL and SHM files from the previous database — they belong to a
	// different database and will corrupt the restored one if left behind.
	for _, ext := range []string{"-wal", "-shm"} {
		path := dbPath + ext
		if err := os.Remove(path); err == nil {
			slog.Info("deleted stale journal file", "file", filepath.Base(path))
		} else if !os.IsNotExist(err) {
			slog.Warn("failed to delete stale journal file", "file", filepath.Base(path), "error", err)
		}
	}

	if err := os.Remove(markerPath); err == nil {
		slog.Info("deleted restore marker", "file", markerFile)
	} else if !os.IsNotExist(err) {
		slog.Warn("failed to delete restore marker", "error", err)
	}

	slog.Info("database restored from snapshot", "snapshot", snapshotName)
	slog.Info("=== RECOVERY INFO ===")
	slog.Info("Pre-restore backup saved, to roll back:",
		"backup", filepath.Base(backupPath),
		"action", fmt.Sprintf("cp %s* %s && rm %s-wal %s-shm && restart", backupPath, dbPath, dbPath, dbPath))
	slog.Info("====================")
	return nil
}

// WriteRestoreMarker records which snapshot should be applied on the next startup.
// The marker file is written next to the DB file so it is on the same filesystem.
func WriteRestoreMarker(dbPath, snapshotDir, id string) error {
	snapshotPath := filepath.Join(snapshotDir, id+".db")
	if _, err := os.Stat(snapshotPath); err != nil {
		slog.Error("snapshot not found for restore marker", "id", id, "path", snapshotPath, "error", err)
		return fmt.Errorf("snapshot %q not found: %w", id, err)
	}
	markerPath := filepath.Join(filepath.Dir(dbPath), markerFile)
	slog.Info("writing restore marker", "marker", markerPath, "snapshot", id)
	return os.WriteFile(markerPath, []byte(id+".db"), 0o644)
}

// Prune deletes the oldest snapshots in snapshotDir, keeping at most keep files.
// A keep value ≤ 0 means unlimited.
func Prune(snapshotDir string, keep int) error {
	if keep <= 0 {
		return nil
	}
	entries, err := os.ReadDir(snapshotDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read snapshot dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".db") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files) // timestamp names sort chronologically

	total := len(files)
	if total <= keep {
		return nil
	}

	toDelete := files[:total-keep]
	slog.Info("pruning old snapshots", "total", total, "keep", keep, "deleting", len(toDelete))
	for _, name := range toDelete {
		path := filepath.Join(snapshotDir, name)
		if err := os.Remove(path); err != nil {
			slog.Warn("failed to delete old snapshot", "file", name, "error", err)
		} else {
			slog.Info("deleted old snapshot", "file", name)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	written, err := io.Copy(out, in)
	if err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	slog.Info("file copied", "src", filepath.Base(src), "dst", filepath.Base(dst), "bytes", written)
	return nil
}
