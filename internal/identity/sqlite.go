package identity

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/saintedlama/degubase/internal/infrastructure/identifier"
	"github.com/saintedlama/degubase/internal/infrastructure/util"
	"github.com/saintedlama/degubase/internal/models"
)

// SQLite is the SQLite adapter for identity.Store.
type SQLite struct {
	db *sql.DB
}

// NewSQLite creates an identity.Store backed by SQLite.
func NewSQLite(db *sql.DB) *SQLite {
	return &SQLite{db: db}
}

// Compile-time check.
var _ Store = (*SQLite)(nil)

// ── Workspaces ────────────────────────────────────────────────────────────────

func (s *SQLite) ListWorkspaces(ctx context.Context) ([]models.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, code, name, context, created_at FROM workspaces ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Workspace
	for rows.Next() {
		var w models.Workspace
		var ca string
		if err := rows.Scan(&w.ID, &w.Code, &w.Name, &w.Context, &ca); err != nil {
			return nil, err
		}
		w.CreatedAt = util.ParseTime(ca)
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *SQLite) CreateWorkspace(ctx context.Context, name, wsCtx, code string) (*models.Workspace, error) {
	cd := newWorkspaceCode(ctx, s.db, code)
	res, err := s.db.ExecContext(ctx, `INSERT INTO workspaces (name, context, code) VALUES (?, ?, ?)`, name, wsCtx, cd)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetWorkspace(ctx, id)
}

func (s *SQLite) GetWorkspace(ctx context.Context, id int64) (*models.Workspace, error) {
	var w models.Workspace
	var ca string
	err := s.db.QueryRowContext(ctx, `SELECT id, code, name, context, created_at FROM workspaces WHERE id = ?`, id).
		Scan(&w.ID, &w.Code, &w.Name, &w.Context, &ca)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	w.CreatedAt = util.ParseTime(ca)
	return &w, nil
}

func (s *SQLite) GetWorkspaceByCode(ctx context.Context, code string) (*models.Workspace, error) {
	var w models.Workspace
	var ca string
	err := s.db.QueryRowContext(ctx, `SELECT id, code, name, context, created_at FROM workspaces WHERE code = ?`, code).
		Scan(&w.ID, &w.Code, &w.Name, &w.Context, &ca)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	w.CreatedAt = util.ParseTime(ca)
	return &w, nil
}

func (s *SQLite) UpdateWorkspace(ctx context.Context, id int64, name, wsCtx string) (*models.Workspace, error) {
	// Code is immutable after creation; only name and context are updated.
	_, err := s.db.ExecContext(ctx, `UPDATE workspaces SET name = ?, context = ? WHERE id = ?`, name, wsCtx, id)
	if err != nil {
		return nil, err
	}
	return s.GetWorkspace(ctx, id)
}

func (s *SQLite) DeleteWorkspace(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM workspaces WHERE id = ?`, id)
	return err
}

// ── Users ─────────────────────────────────────────────────────────────────────

func (s *SQLite) CountUsers(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s *SQLite) CreateUser(ctx context.Context, username, name, passwordHash string, isAdmin bool) (*models.User, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	admin := 0
	if isAdmin {
		admin = 1
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (username, name, password, is_admin, created_at) VALUES (?, ?, ?, ?, ?)`,
		username, name, passwordHash, admin, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetUserByID(ctx, id)
}

func (s *SQLite) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, username, name, is_admin, created_at, password FROM users WHERE username = ?`, username)
	return scanUser(row)
}

func (s *SQLite) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, username, name, is_admin, created_at, password FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func (s *SQLite) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, username, name, is_admin, created_at, password FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

func (s *SQLite) DeleteUser(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

func (s *SQLite) GetUserPasswordHashByUsername(ctx context.Context, username string) (string, error) {
	var hash string
	err := s.db.QueryRowContext(ctx, `SELECT password FROM users WHERE username = ?`, username).Scan(&hash)
	return hash, err
}

// ── Workspace tokens ──────────────────────────────────────────────────────────

func (s *SQLite) CreateWorkspaceToken(ctx context.Context, workspaceID int64, name, tokenHash, prefix string, createdBy *int64) (*models.WorkspaceToken, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO workspace_tokens (workspace_id, name, token_hash, prefix, created_by, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		workspaceID, name, tokenHash, prefix, createdBy, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.getWorkspaceTokenByID(ctx, id)
}

func (s *SQLite) ListWorkspaceTokens(ctx context.Context, workspaceID int64) ([]models.WorkspaceToken, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, workspace_id, name, prefix, created_at FROM workspace_tokens WHERE workspace_id = ? ORDER BY id`,
		workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.WorkspaceToken
	for rows.Next() {
		var t models.WorkspaceToken
		var ca string
		if err := rows.Scan(&t.ID, &t.WorkspaceID, &t.Name, &t.Prefix, &ca); err != nil {
			return nil, err
		}
		t.CreatedAt = util.ParseTime(ca)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *SQLite) GetWorkspaceTokenByHash(ctx context.Context, hash string) (*models.WorkspaceToken, error) {
	var t models.WorkspaceToken
	var ca string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, workspace_id, name, prefix, created_at FROM workspace_tokens WHERE token_hash = ?`, hash).
		Scan(&t.ID, &t.WorkspaceID, &t.Name, &t.Prefix, &ca)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	t.CreatedAt = util.ParseTime(ca)
	return &t, nil
}

func (s *SQLite) DeleteWorkspaceToken(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM workspace_tokens WHERE id = ?`, id)
	return err
}

func (s *SQLite) getWorkspaceTokenByID(ctx context.Context, id int64) (*models.WorkspaceToken, error) {
	var t models.WorkspaceToken
	var ca string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, workspace_id, name, prefix, created_at FROM workspace_tokens WHERE id = ?`, id).
		Scan(&t.ID, &t.WorkspaceID, &t.Name, &t.Prefix, &ca)
	if err != nil {
		return nil, err
	}
	t.CreatedAt = util.ParseTime(ca)
	return &t, nil
}

// ── Scan helpers ──────────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(rs rowScanner) (*models.User, error) {
	var u models.User
	var isAdmin int
	var ca, passwordDiscard string
	if err := rs.Scan(&u.ID, &u.Username, &u.Name, &isAdmin, &ca, &passwordDiscard); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	u.IsAdmin = isAdmin != 0
	u.CreatedAt = util.ParseTime(ca)
	return &u, nil
}

// ── Code helper ────────────────────────────────────────────────────────────────

func newWorkspaceCode(ctx context.Context, db *sql.DB, proposed string) string {
	base := proposed
	if base == "" {
		base = "CODE"
	}
	return identifier.UniqueCode(base, func(cd string) bool {
		var c int
		db.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspaces WHERE code = ?`, cd).Scan(&c)
		return c > 0
	})
}
