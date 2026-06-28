// Package identity manages workspaces, users, authentication, and workspace tokens.
package identity

import (
	"context"

	"github.com/saintedlama/degubase/internal/models"
)

// WorkspaceStore is the workspace persistence contract.
type WorkspaceStore interface {
	ListWorkspaces(ctx context.Context) ([]models.Workspace, error)
	CreateWorkspace(ctx context.Context, name, wsCtx, code string) (*models.Workspace, error)
	GetWorkspace(ctx context.Context, id int64) (*models.Workspace, error)
	GetWorkspaceByCode(ctx context.Context, code string) (*models.Workspace, error)
	UpdateWorkspace(ctx context.Context, id int64, name, wsCtx string, mcpEnabled *bool) (*models.Workspace, error)
	DeleteWorkspace(ctx context.Context, id int64) error
}

// UserStore is the user and credential persistence contract.
type UserStore interface {
	CountUsers(ctx context.Context) (int64, error)
	CreateUser(ctx context.Context, username, name, passwordHash string, isAdmin bool) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByID(ctx context.Context, id int64) (*models.User, error)
	GetUserPasswordHashByUsername(ctx context.Context, username string) (string, error)
	ListUsers(ctx context.Context) ([]models.User, error)
	DeleteUser(ctx context.Context, id int64) error
}

// TokenStore is the workspace API token persistence contract.
type TokenStore interface {
	CreateWorkspaceToken(ctx context.Context, workspaceID int64, name, tokenHash, prefix string, createdBy *int64) (*models.WorkspaceToken, error)
	ListWorkspaceTokens(ctx context.Context, workspaceID int64) ([]models.WorkspaceToken, error)
	GetWorkspaceTokenByHash(ctx context.Context, hash string) (*models.WorkspaceToken, error)
	DeleteWorkspaceToken(ctx context.Context, id int64) error
}

// Store composes all identity-scoped stores.
type Store interface {
	WorkspaceStore
	UserStore
	TokenStore
}
