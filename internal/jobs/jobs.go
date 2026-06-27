package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/saintedlama/degubase/internal/models"
)

// Store persists and queries job runs.
type Store interface {
	StartJob(ctx context.Context, jobName string) (int64, error)
	FinalizeJob(ctx context.Context, id int64, status, outcome, logText string) error
	ListJobRuns(ctx context.Context, limit, offset int) ([]models.JobRun, error)
	CountJobRuns(ctx context.Context) (int64, error)
}

// SQLite is the SQLite adapter for jobs.Store.
type SQLite struct {
	db *sql.DB
}

func NewSQLite(db *sql.DB) *SQLite {
	return &SQLite{db: db}
}

func (s *SQLite) ensureTable() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS job_runs (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		job_name    TEXT    NOT NULL,
		status      TEXT    NOT NULL DEFAULT 'running',
		outcome     TEXT    NOT NULL DEFAULT '',
		log         TEXT    NOT NULL DEFAULT '',
		started_at  TEXT    NOT NULL,
		finished_at TEXT
	)`)
	return err
}

func (s *SQLite) StartJob(ctx context.Context, jobName string) (int64, error) {
	if err := s.ensureTable(); err != nil {
		return 0, fmt.Errorf("ensure job_runs table: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO job_runs (job_name, status, started_at) VALUES (?, 'running', ?)`,
		jobName, now,
	)
	if err != nil {
		return 0, fmt.Errorf("start job: %w", err)
	}
	return res.LastInsertId()
}

func (s *SQLite) FinalizeJob(ctx context.Context, id int64, status, outcome, logText string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx,
		`UPDATE job_runs SET status = ?, outcome = ?, log = ?, finished_at = ? WHERE id = ?`,
		status, outcome, logText, now, id,
	)
	if err != nil {
		return fmt.Errorf("finalize job: %w", err)
	}
	return nil
}

func (s *SQLite) ListJobRuns(ctx context.Context, limit, offset int) ([]models.JobRun, error) {
	if err := s.ensureTable(); err != nil {
		return nil, fmt.Errorf("ensure job_runs table: %w", err)
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, job_name, status, outcome, log, started_at, finished_at
		 FROM job_runs ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list job runs: %w", err)
	}
	defer rows.Close()

	var result []models.JobRun
	for rows.Next() {
		var j models.JobRun
		var startedAt string
		var finishedAtPtr *string
		if err := rows.Scan(&j.ID, &j.JobName, &j.Status, &j.Outcome, &j.Log, &startedAt, &finishedAtPtr); err != nil {
			return nil, fmt.Errorf("scan job run: %w", err)
		}
		j.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
		if finishedAtPtr != nil {
			t, _ := time.Parse(time.RFC3339, *finishedAtPtr)
			j.FinishedAt = &t
		}
		result = append(result, j)
	}
	return result, rows.Err()
}

func (s *SQLite) CountJobRuns(ctx context.Context) (int64, error) {
	if err := s.ensureTable(); err != nil {
		return 0, fmt.Errorf("ensure job_runs table: %w", err)
	}
	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM job_runs`).Scan(&total); err != nil {
		return 0, fmt.Errorf("count job runs: %w", err)
	}
	return total, nil
}
