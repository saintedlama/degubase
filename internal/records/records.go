// Package records manages row CRUD, row history, filtering, sorting, search, and real-time events.
package records

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/saintedlama/degubase/internal/models"
)

// ErrRowNotFound is returned when a row does not exist or does not belong to
// the requested table (preventing cross-table/cross-workspace access).
var ErrRowNotFound = errors.New("row not found")

// RowStore is the row persistence contract.
type RowStore interface {
	ListRows(ctx context.Context, tableID int64, limit, offset int, filters []models.RowFilter, sorts []models.RowSort, columns []models.Column, search string) ([]models.Row, int64, error)
	// ListGroupedRows returns one RowGroup per distinct value of groupByColCode.
	// Each group contains up to limit rows (honouring filters, sorts, and search)
	// and the total count for that group across all pages.
	ListGroupedRows(ctx context.Context, tableID int64, groupByColCode string, limit int, filters []models.RowFilter, sorts []models.RowSort, columns []models.Column, search string) ([]models.RowGroup, error)
	GetRow(ctx context.Context, id int64) (*models.Row, error)
	// columns must be the full column list for the table (used to sync row_links).
	CreateRow(ctx context.Context, tableID int64, data json.RawMessage, columns []models.Column) (*models.Row, error)
	UpdateRow(ctx context.Context, id int64, data json.RawMessage, revisionID string, columns []models.Column) (*models.Row, string, error)
	PatchRow(ctx context.Context, id int64, patch json.RawMessage, revisionID string, columns []models.Column) (*models.Row, string, error)
	// BulkPatch applies patch to all rows matching filters in a single transaction.
	// Returns (updatedRows, count, error). When preview is true no rows are modified
	// and updatedRows is nil; only the count is returned.
	BulkPatch(ctx context.Context, tableID int64, filters []models.RowFilter, patch json.RawMessage, columns []models.Column, preview bool) ([]*models.Row, int64, error)
	DeleteRow(ctx context.Context, id int64) error
	// FindRowLinkReferences returns rows in the workspace that reference targetRowID
	// via a row-link column. Used to block deletes that would leave dangling references.
	FindRowLinkReferences(ctx context.Context, workspaceID, targetRowID int64) ([]models.RowLinkRef, error)
	// ListLinksForRows returns resolved link info for the given source row IDs.
	// Used by the read path to resolve link labels without parsing column options.
	ListLinksForRows(ctx context.Context, sourceRowIDs []int64) ([]RowLinkInfo, error)
	// ListReferencingRows returns paginated rows that reference targetRowID via
	// row-link columns anywhere in the workspace.
	ListReferencingRows(ctx context.Context, workspaceID, targetRowID int64, limit, offset int) ([]ReferencingRowRef, int64, error)
	ListRowHistory(ctx context.Context, rowID int64) ([]models.RowHistory, error)
	CreateAnnotation(ctx context.Context, rowID int64, text string) (*models.RowHistory, error)
	// UpdateAnnotation and DeleteAnnotation scope mutations by rowID so a
	// history entry from another row cannot be modified.
	UpdateAnnotation(ctx context.Context, rowID, historyID int64, text string) (*models.RowHistory, error)
	DeleteAnnotation(ctx context.Context, rowID, historyID int64) error
	UpdateChangeAnnotation(ctx context.Context, rowID, historyID int64, text string) (*models.RowHistory, error)
}

// Store is the records-scoped store.
type Store interface {
	RowStore
}

// ColumnLister is the narrow schema dependency needed for row filtering and sorting.
type ColumnLister interface {
	ListColumns(ctx context.Context, tableID int64) ([]models.Column, error)
}

// RowLinkInfo is a resolved row-link entry for read-path label resolution.
// Codes are included so the caller can access code-keyed row data directly.
type RowLinkInfo struct {
	SourceRowID   int64
	SourceColCode string
	TargetRowID   int64
	TargetTableID int64
	TargetColCode string // empty when target_col_id is NULL (fall back to old resolution)
}

// ReferencingRowRef is a raw row from the row_links table before label
// resolution. The handler resolves SourceRowLabel from SourceRowData.
type ReferencingRowRef struct {
	SourceRowID     int64
	SourceTableID   int64
	SourceTableCode string
	SourceTableName string
	ViaColumnCode   string
	ViaColumnName   string
	// SourceRowData is the raw JSON data of the source row, used to resolve
	// the label in the handler layer.
	SourceRowData string
}
