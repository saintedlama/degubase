package skills

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/saintedlama/degubase/internal/infrastructure/util"
	"github.com/saintedlama/degubase/internal/models"
)

type rowScanner interface {
	Scan(dest ...any) error
}

type SQLite struct {
	db *sql.DB
}

func NewSQLite(db *sql.DB) *SQLite {
	return &SQLite{db: db}
}

var _ Store = (*SQLite)(nil)

func (s *SQLite) ListSkills(ctx context.Context, workspaceID int64) ([]models.Skill, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, name, description, table_ids, operations, token_id, created_at
		FROM skills WHERE workspace_id = ? ORDER BY created_at DESC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Skill
	for rows.Next() {
		sk, err := scanSkill(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sk)
	}
	return out, rows.Err()
}

func (s *SQLite) CreateSkill(ctx context.Context, workspaceID int64, name, description string, tableIDs []int64, operations []string, tokenID *int64) (*models.Skill, error) {
	tableIDsJSON, err := json.Marshal(tableIDs)
	if err != nil {
		return nil, err
	}
	if len(operations) == 0 {
		operations = []string{"read", "create", "update", "delete"}
	}
	opsJSON, err := json.Marshal(operations)
	if err != nil {
		return nil, err
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO skills (workspace_id, name, description, table_ids, operations, token_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`, workspaceID, name, description, string(tableIDsJSON), string(opsJSON), tokenID)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetSkill(ctx, id)
}

func (s *SQLite) UpdateSkill(ctx context.Context, id int64, name, description string, tableIDs []int64, operations []string) (*models.Skill, error) {
	tableIDsJSON, err := json.Marshal(tableIDs)
	if err != nil {
		return nil, err
	}
	if len(operations) == 0 {
		operations = []string{"read", "create", "update", "delete"}
	}
	opsJSON, err := json.Marshal(operations)
	if err != nil {
		return nil, err
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE skills SET name=?, description=?, table_ids=?, operations=? WHERE id=?
	`, name, description, string(tableIDsJSON), string(opsJSON), id)
	if err != nil {
		return nil, err
	}
	return s.GetSkill(ctx, id)
}

func (s *SQLite) GetSkill(ctx context.Context, id int64) (*models.Skill, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, name, description, table_ids, operations, token_id, created_at
		FROM skills WHERE id = ?
	`, id)
	return scanSkillRow(row)
}

func (s *SQLite) DeleteSkill(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM skills WHERE id = ?`, id)
	return err
}

func scanSkill(rs rowScanner) (*models.Skill, error) {
	var sk models.Skill
	var tableIDsJSON, opsJSON string
	var tokenID sql.NullInt64
	var createdAt string
	if err := rs.Scan(&sk.ID, &sk.WorkspaceID, &sk.Name, &sk.Description, &tableIDsJSON, &opsJSON, &tokenID, &createdAt); err != nil {
		return nil, err
	}
	if tokenID.Valid {
		sk.TokenID = &tokenID.Int64
	}
	if err := json.Unmarshal([]byte(tableIDsJSON), &sk.TableIDs); err != nil {
		sk.TableIDs = []int64{}
	}
	if err := json.Unmarshal([]byte(opsJSON), &sk.Operations); err != nil {
		sk.Operations = []string{"read", "create", "update", "delete"}
	}
	sk.CreatedAt = util.ParseTime(createdAt)
	return &sk, nil
}

func scanSkillRow(r *sql.Row) (*models.Skill, error) {
	var sk models.Skill
	var tableIDsJSON, opsJSON string
	var tokenID sql.NullInt64
	var createdAt string
	if err := r.Scan(&sk.ID, &sk.WorkspaceID, &sk.Name, &sk.Description, &tableIDsJSON, &opsJSON, &tokenID, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if tokenID.Valid {
		sk.TokenID = &tokenID.Int64
	}
	if err := json.Unmarshal([]byte(tableIDsJSON), &sk.TableIDs); err != nil {
		sk.TableIDs = []int64{}
	}
	if err := json.Unmarshal([]byte(opsJSON), &sk.Operations); err != nil {
		sk.Operations = []string{"read", "create", "update", "delete"}
	}
	sk.CreatedAt = util.ParseTime(createdAt)
	return &sk, nil
}
