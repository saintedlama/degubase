package snapshots

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/saintedlama/degubase/internal/jobs"
)

// ScheduledJob returns a function that creates a snapshot and records the run via
// the job store. Intended for use with a scheduler (gocron, cron, etc.).
func ScheduledJob(db *sql.DB, snapshotDir string, maxSnapshots int, jobStore jobs.Store) func() {
	return func() {
		ctx := context.Background()
		jobID, _ := jobStore.StartJob(ctx, "scheduled-snapshot")

		var status, outcome, logText string
		maxKeep := maxSnapshots
		if maxKeep == 0 {
			maxKeep = DefaultMaxSnapshots
		}

		snap, err := Create(db, snapshotDir)
		if err != nil {
			status = "failed"
			outcome = "Snapshot failed"
			logText = err.Error()
			slog.Error("scheduled snapshot failed", "error", err)
		} else {
			status = "completed"
			outcome = fmt.Sprintf("Snapshot %s created (%d bytes)", snap.ID, snap.SizeBytes)
			logText = fmt.Sprintf("snapshot_id=%s size_bytes=%d", snap.ID, snap.SizeBytes)
			slog.Info("scheduled snapshot created", "id", snap.ID, "size_bytes", snap.SizeBytes)
			if err := Prune(snapshotDir, maxKeep); err != nil {
				logText += fmt.Sprintf("\nprune error: %v", err)
				slog.Warn("scheduled snapshot pruning failed", "error", err)
			}
		}

		if jobID > 0 {
			jobStore.FinalizeJob(ctx, jobID, status, outcome, logText)
		}
	}
}
