package schema

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/saintedlama/degubase/internal/infrastructure/identifier"
	"github.com/saintedlama/degubase/internal/infrastructure/util"
	"github.com/saintedlama/degubase/internal/models"
)

// SQLite is the SQLite adapter for schema.Store.
type SQLite struct {
	db *sql.DB
}

// NewSQLite creates a schema.Store backed by SQLite.
func NewSQLite(db *sql.DB) *SQLite {
	return &SQLite{db: db}
}

var _ Store = (*SQLite)(nil)

// ── Tables ────────────────────────────────────────────────────────────────────

func (s *SQLite) ListTables(ctx context.Context, workspaceID int64) ([]models.Table, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, code, workspace_id, name, context, icon, default_view_id, created_at FROM tables WHERE workspace_id = ? ORDER BY id`,
		workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Table
	for rows.Next() {
		t, err := scanTable(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (s *SQLite) CreateTable(ctx context.Context, workspaceID int64, name, tableCtx, icon, code string) (*models.Table, error) {
	cd := newTableCode(ctx, s.db, workspaceID, code)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx,
		`INSERT INTO tables (workspace_id, name, context, icon, code) VALUES (?, ?, ?, ?, ?)`,
		workspaceID, name, tableCtx, icon, cd)
	if err != nil {
		return nil, err
	}
	tableID, _ := res.LastInsertId()
	var viewID int64
	if r, err := tx.ExecContext(ctx,
		`INSERT INTO views (table_id, name, type, is_default, code) VALUES (?, 'Table', 'tabular', 1, 'default-view')`,
		tableID); err != nil {
		return nil, err
	} else {
		viewID, _ = r.LastInsertId()
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE tables SET default_view_id = ? WHERE id = ?`, viewID, tableID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetTable(ctx, tableID)
}

func (s *SQLite) GetTable(ctx context.Context, id int64) (*models.Table, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, code, workspace_id, name, context, icon, default_view_id, created_at FROM tables WHERE id = ?`, id)
	t, err := scanTableRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

func (s *SQLite) GetTableByCode(ctx context.Context, workspaceID int64, code string) (*models.Table, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, code, workspace_id, name, context, icon, default_view_id, created_at FROM tables WHERE workspace_id = ? AND code = ?`,
		workspaceID, code)
	t, err := scanTableRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

func (s *SQLite) UpdateTable(ctx context.Context, id int64, name, tableCtx, icon string, defaultViewID *int64) (*models.Table, error) {
	// Code is immutable after creation; only name, context, icon, and default_view_id are updated.
	_, err := s.db.ExecContext(ctx,
		`UPDATE tables SET name = ?, context = ?, icon = ?, default_view_id = ? WHERE id = ?`, name, tableCtx, icon, defaultViewID, id)
	if err != nil {
		return nil, err
	}
	return s.GetTable(ctx, id)
}

func (s *SQLite) DeleteTable(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tables WHERE id = ?`, id)
	return err
}

// ── Columns ───────────────────────────────────────────────────────────────────

func (s *SQLite) ListColumns(ctx context.Context, tableID int64) ([]models.Column, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, table_id, name, type, options, position, code FROM columns WHERE table_id = ? ORDER BY position, id`,
		tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Column
	for rows.Next() {
		c, err := scanColumn(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// ErrCodeConflict is returned when a new column name would produce a code that
// already exists on the same table.
var ErrCodeConflict = fmt.Errorf("a field with that name already exists")

func (s *SQLite) CreateColumn(ctx context.Context, tableID int64, name string, colType models.ColumnType, options json.RawMessage) (*models.Column, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	colCode := identifier.Make(name)
	var taken int
	_ = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM columns WHERE table_id = ? AND code = ?`, tableID, colCode).Scan(&taken)
	if taken > 0 {
		return nil, ErrCodeConflict
	}

	var pos int
	_ = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(position), -1) + 1 FROM columns WHERE table_id = ?`, tableID).Scan(&pos)

	optStr := OptionsToString(options)
	res, err := tx.ExecContext(ctx,
		`INSERT INTO columns (table_id, name, type, options, position, code) VALUES (?, ?, ?, ?, ?, ?)`,
		tableID, name, string(colType), optStr, pos, colCode)
	if err != nil {
		return nil, err
	}
	colID, _ := res.LastInsertId()

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	var col models.Column
	var optStr2 sql.NullString
	err = s.db.QueryRowContext(ctx,
		`SELECT id, table_id, name, type, options, position, code FROM columns WHERE id = ?`, colID).
		Scan(&col.ID, &col.TableID, &col.Name, (*string)(&col.Type), &optStr2, &col.Position, &col.Code)
	if err != nil {
		return nil, err
	}
	if optStr2.Valid {
		col.Options = json.RawMessage(optStr2.String)
	}
	return &col, nil
}

func (s *SQLite) GetColumn(ctx context.Context, id int64) (*models.Column, error) {
	var col models.Column
	var optStr sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, table_id, name, type, options, position, code FROM columns WHERE id = ?`, id).
		Scan(&col.ID, &col.TableID, &col.Name, (*string)(&col.Type), &optStr, &col.Position, &col.Code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if optStr.Valid {
		col.Options = json.RawMessage(optStr.String)
	}
	return &col, nil
}

func (s *SQLite) GetColumnByCode(ctx context.Context, tableID int64, code string) (*models.Column, error) {
	var col models.Column
	var optStr sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, table_id, name, type, options, position, code FROM columns WHERE table_id = ? AND code = ?`, tableID, code).
		Scan(&col.ID, &col.TableID, &col.Name, (*string)(&col.Type), &optStr, &col.Position, &col.Code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if optStr.Valid {
		col.Options = json.RawMessage(optStr.String)
	}
	return &col, nil
}

func (s *SQLite) UpdateColumn(ctx context.Context, id int64, name string, colType models.ColumnType, options json.RawMessage) (*models.Column, error) {
	optStr := OptionsToString(options)
	// Code is frozen at creation time and never updated on rename.
	_, err := s.db.ExecContext(ctx,
		`UPDATE columns SET name = ?, type = ?, options = ? WHERE id = ?`, name, string(colType), optStr, id)
	if err != nil {
		return nil, err
	}
	return s.GetColumn(ctx, id)
}

func (s *SQLite) ReorderColumns(ctx context.Context, tableID int64, ids []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx,
			`UPDATE columns SET position = ? WHERE id = ? AND table_id = ?`, i, id, tableID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLite) DeleteColumn(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM columns WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("column not found")
	}
	return nil
}

// ── Views ─────────────────────────────────────────────────────────────────────

func (s *SQLite) ListViews(ctx context.Context, tableID int64) ([]models.View, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, code, table_id, name, type, config, created_at FROM views WHERE table_id = ? ORDER BY id`,
		tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.View
	for rows.Next() {
		v, err := scanView(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (s *SQLite) CreateView(ctx context.Context, tableID int64, name string, viewType models.ViewType, config *models.ViewConfig) (*models.View, error) {
	sl := newViewCode(ctx, s.db, tableID, name)
	cfgStr := viewConfigToString(config)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO views (table_id, name, type, config, is_default, code) VALUES (?, ?, ?, ?, 0, ?)`,
		tableID, name, string(viewType), cfgStr, sl)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetView(ctx, id)
}

func (s *SQLite) GetView(ctx context.Context, id int64) (*models.View, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, code, table_id, name, type, config, created_at FROM views WHERE id = ?`, id)
	v, err := scanViewRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return v, err
}

func (s *SQLite) GetViewByCode(ctx context.Context, tableID int64, code string) (*models.View, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, code, table_id, name, type, config, created_at FROM views WHERE table_id = ? AND code = ?`,
		tableID, code)
	v, err := scanViewRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return v, err
}

func (s *SQLite) UpdateView(ctx context.Context, id int64, name string, config *models.ViewConfig) (*models.View, error) {
	var tableID int64
	if err := s.db.QueryRowContext(ctx, `SELECT table_id FROM views WHERE id = ?`, id).Scan(&tableID); err != nil {
		return nil, fmt.Errorf("view not found")
	}
	code := updateViewCode(ctx, s.db, tableID, name, id)
	cfgStr := viewConfigToString(config)
	_, err := s.db.ExecContext(ctx,
		`UPDATE views SET name = ?, config = ?, code = ? WHERE id = ?`, name, cfgStr, code, id)
	if err != nil {
		return nil, err
	}
	return s.GetView(ctx, id)
}

func (s *SQLite) DeleteView(ctx context.Context, id int64) error {
	var tableID int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT table_id FROM views WHERE id = ?`, id).Scan(&tableID); err != nil {
		return fmt.Errorf("view not found")
	}
	var defaultViewID sql.NullInt64
	s.db.QueryRowContext(ctx,
		`SELECT default_view_id FROM tables WHERE id = ?`, tableID).Scan(&defaultViewID)
	if defaultViewID.Valid && defaultViewID.Int64 == id {
		return fmt.Errorf("view not found or is the default view")
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM views WHERE id = ?`, id)
	return err
}

// ── Code helpers ─────────────────────────────────────────────────────────────

func newTableCode(ctx context.Context, db *sql.DB, workspaceID int64, proposed string) string {
	base := proposed
	if base == "" {
		base = "CODE"
	}
	return identifier.UniqueCode(base, func(cd string) bool {
		var c int
		db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tables WHERE workspace_id = ? AND code = ?`, workspaceID, cd).Scan(&c)
		return c > 0
	})
}

func newViewCode(ctx context.Context, db *sql.DB, tableID int64, name string) string {
	base := identifier.Make(name)
	return identifier.Unique(base, func(cd string) bool {
		var c int
		db.QueryRowContext(ctx, `SELECT COUNT(*) FROM views WHERE table_id = ? AND code = ?`, tableID, cd).Scan(&c)
		return c > 0
	})
}

func updateViewCode(ctx context.Context, db *sql.DB, tableID int64, name string, excludeID int64) string {
	base := identifier.Make(name)
	return identifier.Unique(base, func(cd string) bool {
		var c int
		db.QueryRowContext(ctx, `SELECT COUNT(*) FROM views WHERE table_id = ? AND code = ? AND id != ?`, tableID, cd, excludeID).Scan(&c)
		return c > 0
	})
}

// ── Scan helpers (local) ──────────────────────────────────────────────────────

func scanTable(rs interface{ Scan(...any) error }) (*models.Table, error) {
	var t models.Table
	var ca string
	var dvi sql.NullInt64
	if err := rs.Scan(&t.ID, &t.Code, &t.WorkspaceID, &t.Name, &t.Context, &t.Icon, &dvi, &ca); err != nil {
		return nil, err
	}
	if dvi.Valid {
		t.DefaultViewID = &dvi.Int64
	}
	t.CreatedAt = util.ParseTime(ca)
	return &t, nil
}

func scanTableRow(r *sql.Row) (*models.Table, error) {
	var t models.Table
	var ca string
	var dvi sql.NullInt64
	if err := r.Scan(&t.ID, &t.Code, &t.WorkspaceID, &t.Name, &t.Context, &t.Icon, &dvi, &ca); err != nil {
		return nil, err
	}
	if dvi.Valid {
		t.DefaultViewID = &dvi.Int64
	}
	t.CreatedAt = util.ParseTime(ca)
	return &t, nil
}

func scanColumn(rs interface{ Scan(...any) error }) (*models.Column, error) {
	var c models.Column
	var optStr sql.NullString
	if err := rs.Scan(&c.ID, &c.TableID, &c.Name, (*string)(&c.Type), &optStr, &c.Position, &c.Code); err != nil {
		return nil, err
	}
	if optStr.Valid {
		c.Options = json.RawMessage(optStr.String)
	}
	return &c, nil
}

func scanView(rs interface{ Scan(...any) error }) (*models.View, error) {
	var v models.View
	var cfgStr sql.NullString
	var ca string
	if err := rs.Scan(&v.ID, &v.Code, &v.TableID, &v.Name, (*string)(&v.Type), &cfgStr, &ca); err != nil {
		return nil, err
	}
	if cfgStr.Valid {
		var cfg models.ViewConfig
		if json.Unmarshal([]byte(cfgStr.String), &cfg) == nil {
			v.Config = &cfg
		}
	}
	v.CreatedAt = util.ParseTime(ca)
	return &v, nil
}

func scanViewRow(r *sql.Row) (*models.View, error) {
	var v models.View
	var cfgStr sql.NullString
	var ca string
	if err := r.Scan(&v.ID, &v.Code, &v.TableID, &v.Name, (*string)(&v.Type), &cfgStr, &ca); err != nil {
		return nil, err
	}
	if cfgStr.Valid {
		var cfg models.ViewConfig
		if json.Unmarshal([]byte(cfgStr.String), &cfg) == nil {
			v.Config = &cfg
		}
	}
	v.CreatedAt = util.ParseTime(ca)
	return &v, nil
}

// OptionsToString converts a JSON raw message to a nullable string for SQL.
func OptionsToString(options json.RawMessage) sql.NullString {
	if len(options) == 0 || string(options) == "null" {
		return sql.NullString{}
	}
	return sql.NullString{String: string(options), Valid: true}
}

func viewConfigToString(config *models.ViewConfig) sql.NullString {
	if config == nil {
		return sql.NullString{}
	}
	b, err := json.Marshal(config)
	if err != nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(b), Valid: true}
}
