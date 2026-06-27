// Package schema manages tables, columns, and views — the structure of data.
package schema

import (
	"context"
	"encoding/json"

	"github.com/saintedlama/degubase/internal/models"
)

// TableStore is the table persistence contract.
type TableStore interface {
	ListTables(ctx context.Context, workspaceID int64) ([]models.Table, error)
	CreateTable(ctx context.Context, workspaceID int64, name, tableCtx, icon, code string) (*models.Table, error)
	GetTable(ctx context.Context, id int64) (*models.Table, error)
	GetTableByCode(ctx context.Context, workspaceID int64, code string) (*models.Table, error)
	UpdateTable(ctx context.Context, id int64, name, tableCtx, icon string, defaultViewID *int64) (*models.Table, error)
	DeleteTable(ctx context.Context, id int64) error
}

// ColumnStore is the column persistence contract.
type ColumnStore interface {
	ListColumns(ctx context.Context, tableID int64) ([]models.Column, error)
	CreateColumn(ctx context.Context, tableID int64, name string, colType models.ColumnType, options json.RawMessage) (*models.Column, error)
	GetColumn(ctx context.Context, id int64) (*models.Column, error)
	GetColumnByCode(ctx context.Context, tableID int64, code string) (*models.Column, error)
	UpdateColumn(ctx context.Context, id int64, name string, colType models.ColumnType, options json.RawMessage) (*models.Column, error)
	ReorderColumns(ctx context.Context, tableID int64, ids []int64) error
	DeleteColumn(ctx context.Context, id int64) error
}

// ViewStore is the view persistence contract.
type ViewStore interface {
	ListViews(ctx context.Context, tableID int64) ([]models.View, error)
	CreateView(ctx context.Context, tableID int64, name string, viewType models.ViewType, config *models.ViewConfig) (*models.View, error)
	GetView(ctx context.Context, id int64) (*models.View, error)
	GetViewByCode(ctx context.Context, tableID int64, code string) (*models.View, error)
	UpdateView(ctx context.Context, id int64, name string, config *models.ViewConfig) (*models.View, error)
	DeleteView(ctx context.Context, id int64) error
}

// Store composes all schema-scoped stores.
type Store interface {
	TableStore
	ColumnStore
	ViewStore
}
