// Package skills manages skill CRUD and markdown generation.
package skills

import (
	"context"

	"github.com/saintedlama/degubase/internal/models"
)

// TableReader is the narrow schema dependency needed for skill generation.
type TableReader interface {
	GetTable(ctx context.Context, id int64) (*models.Table, error)
	ListColumns(ctx context.Context, tableID int64) ([]models.Column, error)
}

// RowReader allows skill generation to fetch a small sample of rows.
type RowReader interface {
	ListRows(ctx context.Context, tableID int64, limit, offset int, filters []models.RowFilter, sorts []models.RowSort, columns []models.Column, search string) ([]models.Row, int64, error)
}

// WorkspaceTokenCreator is the narrow identity dependency needed for skill creation.
type WorkspaceTokenCreator interface {
	GetWorkspaceByCode(ctx context.Context, code string) (*models.Workspace, error)
	CreateWorkspaceToken(ctx context.Context, workspaceID int64, name, tokenHash, prefix string, createdBy *int64) (*models.WorkspaceToken, error)
}

// Store is the skill persistence contract.
type Store interface {
	ListSkills(ctx context.Context, workspaceID int64) ([]models.Skill, error)
	CreateSkill(ctx context.Context, workspaceID int64, name, description string, tableIDs []int64, operations []string, tokenID *int64) (*models.Skill, error)
	UpdateSkill(ctx context.Context, id int64, name, description string, tableIDs []int64, operations []string) (*models.Skill, error)
	GetSkill(ctx context.Context, id int64) (*models.Skill, error)
	DeleteSkill(ctx context.Context, id int64) error
}
