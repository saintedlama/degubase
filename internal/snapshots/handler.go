package snapshots

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
)

const DefaultMaxSnapshots = 7

// Handler serves the snapshot and restore endpoints.
type Handler struct {
	DB           *sql.DB
	DBPath       string
	SnapshotDir  string
	MaxSnapshots int // maximum snapshots to retain; 0 uses DefaultMaxSnapshots
	// Shutdown is called after the restore marker is written. Defaults to os.Exit(0).
	Shutdown func()
}

// ListSnapshots returns all snapshots, newest first.
func (h *Handler) ListSnapshots(w http.ResponseWriter, r *http.Request) {
	snaps, err := List(h.SnapshotDir)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, http.StatusOK, snaps)
}

// CreateSnapshot creates a new snapshot of the live database and prunes old ones.
func (h *Handler) CreateSnapshot(w http.ResponseWriter, r *http.Request) {
	slog.Info("snapshot create requested")
	snap, err := Create(h.DB, h.SnapshotDir)
	if err != nil {
		slog.Error("snapshot create failed", "error", err)
		httplib.InternalErr(w, err)
		return
	}
	keep := h.MaxSnapshots
	if keep == 0 {
		keep = DefaultMaxSnapshots
	}
	if err := Prune(h.SnapshotDir, keep); err != nil {
		slog.Warn("snapshot pruning failed", "error", err)
	}
	slog.Info("snapshot create completed", "id", snap.ID, "size_bytes", snap.SizeBytes)
	httplib.Respond(w, http.StatusCreated, snap)
}

// ScheduleRestore writes the restore marker for the given snapshot ID and shuts
// down the process so Docker's restart policy can apply the restore on startup.
func (h *Handler) ScheduleRestore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	slog.Info("snapshot restore requested", "snapshot", id)
	if err := WriteRestoreMarker(h.DBPath, h.SnapshotDir, id); err != nil {
		slog.Error("restore marker write failed", "snapshot", id, "error", err)
		httplib.InternalErr(w, err)
		return
	}
	slog.Info("restore scheduled, initiating shutdown", "snapshot", id)
	httplib.Respond(w, http.StatusAccepted, map[string]string{
		"message": "restore scheduled; server is restarting",
	})
	shutdown := h.Shutdown
	if shutdown == nil {
		shutdown = func() { os.Exit(0) }
	}
	go func() {
		time.Sleep(100 * time.Millisecond) // let the response flush
		slog.Info("shutting down for restore", "snapshot", id)
		shutdown()
	}()
}
