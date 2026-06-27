package automation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/saintedlama/degubase/internal/infrastructure/util"
	"github.com/saintedlama/degubase/internal/models"
)

type rowScanner interface {
	Scan(dest ...any) error
}

type txExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type SQLite struct {
	db *sql.DB
}

func NewSQLite(db *sql.DB) *SQLite {
	return &SQLite{db: db}
}

var _ Store = (*SQLite)(nil)

// ── Scripts ───────────────────────────────────────────────────────────────────

func (s *SQLite) ListScripts(ctx context.Context, workspaceID int64) ([]models.Script, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, workspace_id, name, event_type, code, enabled, created_at, updated_at FROM scripts WHERE workspace_id = ? ORDER BY id",
		workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Script
	for rows.Next() {
		sc, err := scanScript(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if err := s.loadScriptTableIDs(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *SQLite) CreateScript(ctx context.Context, workspaceID int64, tableIDs []int64, name string, eventType models.ScriptEventType, code string, enabled bool) (*models.Script, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	en := 0
	if enabled {
		en = 1
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx,
		"INSERT INTO scripts (workspace_id, name, event_type, code, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		workspaceID, name, string(eventType), code, en, now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	if err := insertScriptTables(ctx, tx, id, tableIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetScript(ctx, id)
}

func (s *SQLite) GetScript(ctx context.Context, id int64) (*models.Script, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id, workspace_id, name, event_type, code, enabled, created_at, updated_at FROM scripts WHERE id = ?", id)
	sc, err := scanScriptRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := s.loadScriptTableIDs(ctx, sc); err != nil {
		return nil, err
	}
	return sc, nil
}

func (s *SQLite) UpdateScript(ctx context.Context, id int64, tableIDs []int64, name string, eventType models.ScriptEventType, code string, enabled bool) (*models.Script, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	en := 0
	if enabled {
		en = 1
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		"UPDATE scripts SET name = ?, event_type = ?, code = ?, enabled = ?, updated_at = ? WHERE id = ?",
		name, string(eventType), code, en, now, id); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM script_tables WHERE script_id = ?", id); err != nil {
		return nil, err
	}
	if err := insertScriptTables(ctx, tx, id, tableIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetScript(ctx, id)
}

func (s *SQLite) DeleteScript(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM scripts WHERE id = ?", id)
	return err
}

func (s *SQLite) loadScriptTableIDs(ctx context.Context, sc *models.Script) error {
	rows, err := s.db.QueryContext(ctx, "SELECT table_id FROM script_tables WHERE script_id = ? ORDER BY table_id", sc.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	sc.TableIDs = []int64{}
	for rows.Next() {
		var tid int64
		if err := rows.Scan(&tid); err != nil {
			return err
		}
		sc.TableIDs = append(sc.TableIDs, tid)
	}
	return rows.Err()
}

func insertScriptTables(ctx context.Context, tx txExecer, scriptID int64, tableIDs []int64) error {
	for _, tid := range tableIDs {
		if _, err := tx.ExecContext(ctx, "INSERT INTO script_tables (script_id, table_id) VALUES (?, ?)", scriptID, tid); err != nil {
			return err
		}
	}
	return nil
}

func scanScript(rs rowScanner) (*models.Script, error) {
	var sc models.Script
	var en int
	var ca, ua string
	if err := rs.Scan(&sc.ID, &sc.WorkspaceID, &sc.Name, (*string)(&sc.EventType), &sc.Code, &en, &ca, &ua); err != nil {
		return nil, err
	}
	sc.Enabled = en != 0
	sc.CreatedAt = util.ParseTime(ca)
	sc.UpdatedAt = util.ParseTime(ua)
	return &sc, nil
}

func scanScriptRow(r *sql.Row) (*models.Script, error) {
	var sc models.Script
	var en int
	var ca, ua string
	if err := r.Scan(&sc.ID, &sc.WorkspaceID, &sc.Name, (*string)(&sc.EventType), &sc.Code, &en, &ca, &ua); err != nil {
		return nil, err
	}
	sc.Enabled = en != 0
	sc.CreatedAt = util.ParseTime(ca)
	sc.UpdatedAt = util.ParseTime(ua)
	return &sc, nil
}

// ── Script env vars ───────────────────────────────────────────────────────────

func (s *SQLite) ListScriptEnvVars(ctx context.Context, workspaceID int64) ([]models.ScriptEnvVar, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, workspace_id, key, value, is_secret FROM script_env_vars WHERE workspace_id = ? ORDER BY key",
		workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ScriptEnvVar
	for rows.Next() {
		e, err := scanEnvVar(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

func (s *SQLite) UpsertScriptEnvVar(ctx context.Context, workspaceID int64, key, value string, isSecret bool) (*models.ScriptEnvVar, error) {
	sec := 0
	if isSecret {
		sec = 1
	}
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO script_env_vars (workspace_id, key, value, is_secret) VALUES (?, ?, ?, ?) ON CONFLICT(workspace_id, key) DO UPDATE SET value = CASE WHEN excluded.value = '' THEN script_env_vars.value ELSE excluded.value END, is_secret = script_env_vars.is_secret",
		workspaceID, key, value, sec)
	if err != nil {
		return nil, err
	}
	var e models.ScriptEnvVar
	var isSecInt int
	err = s.db.QueryRowContext(ctx,
		"SELECT id, workspace_id, key, value, is_secret FROM script_env_vars WHERE workspace_id = ? AND key = ?",
		workspaceID, key).Scan(&e.ID, &e.WorkspaceID, &e.Key, &e.Value, &isSecInt)
	if err != nil {
		return nil, err
	}
	e.IsSecret = isSecInt != 0
	return &e, nil
}

func scanEnvVar(rs rowScanner) (*models.ScriptEnvVar, error) {
	var e models.ScriptEnvVar
	var isSecInt int
	if err := rs.Scan(&e.ID, &e.WorkspaceID, &e.Key, &e.Value, &isSecInt); err != nil {
		return nil, err
	}
	e.IsSecret = isSecInt != 0
	return &e, nil
}

func (s *SQLite) DeleteScriptEnvVar(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM script_env_vars WHERE id = ?", id)
	return err
}

// ── Script executions ─────────────────────────────────────────────────────────

func (s *SQLite) ListScriptExecutions(ctx context.Context, scriptID int64, limit int) ([]models.ScriptExecution, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, script_id, row_id, table_code, event_type, started_at, ended_at, duration_ms, success, logs, depth, triggered_by_execution_id FROM script_executions WHERE script_id = ? ORDER BY id DESC LIMIT ?",
		scriptID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ScriptExecution
	for rows.Next() {
		ex, err := scanExecution(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *ex)
	}
	return out, rows.Err()
}

func (s *SQLite) CreateScriptExecution(ctx context.Context, exec models.ScriptExecution) (*models.ScriptExecution, error) {
	logsJSON, err := json.Marshal(exec.Logs)
	if err != nil {
		return nil, err
	}
	suc := 0
	if exec.Success {
		suc = 1
	}
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO script_executions (script_id, row_id, table_code, event_type, started_at, ended_at, duration_ms, success, logs, depth, triggered_by_execution_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		exec.ScriptID, exec.RowID, exec.TableCode,
		string(exec.EventType),
		exec.StartedAt.UTC().Format(time.RFC3339),
		exec.EndedAt.UTC().Format(time.RFC3339),
		exec.DurationMs, suc, string(logsJSON),
		exec.Depth, exec.TriggeredByID)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	row := s.db.QueryRowContext(ctx,
		"SELECT id, script_id, row_id, table_code, event_type, started_at, ended_at, duration_ms, success, logs, depth, triggered_by_execution_id FROM script_executions WHERE id = ?", id)
	return scanExecutionRow(row)
}

func scanExecution(rs rowScanner) (*models.ScriptExecution, error) {
	var ex models.ScriptExecution
	var rowID, triggeredByID sql.NullInt64
	var suc int
	var sa, ea, logsStr string
	if err := rs.Scan(&ex.ID, &ex.ScriptID, &rowID, &ex.TableCode, (*string)(&ex.EventType), &sa, &ea, &ex.DurationMs, &suc, &logsStr, &ex.Depth, &triggeredByID); err != nil {
		return nil, err
	}
	if rowID.Valid {
		ex.RowID = &rowID.Int64
	}
	if triggeredByID.Valid {
		ex.TriggeredByID = &triggeredByID.Int64
	}
	ex.Success = suc != 0
	ex.StartedAt = util.ParseTime(sa)
	ex.EndedAt = util.ParseTime(ea)
	if err := json.Unmarshal([]byte(logsStr), &ex.Logs); err != nil {
		ex.Logs = []models.ScriptLogEntry{}
	}
	return &ex, nil
}

func scanExecutionRow(r *sql.Row) (*models.ScriptExecution, error) {
	var ex models.ScriptExecution
	var rowID, triggeredByID sql.NullInt64
	var suc int
	var sa, ea, logsStr string
	if err := r.Scan(&ex.ID, &ex.ScriptID, &rowID, &ex.TableCode, (*string)(&ex.EventType), &sa, &ea, &ex.DurationMs, &suc, &logsStr, &ex.Depth, &triggeredByID); err != nil {
		return nil, err
	}
	if rowID.Valid {
		ex.RowID = &rowID.Int64
	}
	if triggeredByID.Valid {
		ex.TriggeredByID = &triggeredByID.Int64
	}
	ex.Success = suc != 0
	ex.StartedAt = util.ParseTime(sa)
	ex.EndedAt = util.ParseTime(ea)
	if err := json.Unmarshal([]byte(logsStr), &ex.Logs); err != nil {
		ex.Logs = []models.ScriptLogEntry{}
	}
	return &ex, nil
}
