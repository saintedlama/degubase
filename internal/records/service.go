package records

import (
	"context"
	"encoding/json"

	"github.com/saintedlama/degubase/internal/models"
)

// RowService wraps the store and handles code↔ID translation transparently.
// Handlers call the service with code-keyed data; the service translates to/from
// ID-keyed data for the store.
type RowService struct {
	Store Store
	Cols  ColumnLister
}

func (s *RowService) colMaps(ctx context.Context, tableID int64) (colMaps, []models.Column, error) {
	cols, err := s.Cols.ListColumns(ctx, tableID)
	if err != nil {
		return colMaps{}, nil, err
	}
	columns := models.AppendVirtualColumns(tableID, cols)
	return buildColMaps(columns), columns, nil
}

func (s *RowService) ListGroupedRows(ctx context.Context, tableID int64, groupByColCode string, limit int, filters []models.RowFilter, sorts []models.RowSort, search string) ([]models.RowGroup, error) {
	cm, columns, err := s.colMaps(ctx, tableID)
	if err != nil {
		return nil, err
	}
	groups, err := s.Store.ListGroupedRows(ctx, tableID, groupByColCode, limit, filters, sorts, columns, search)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		for j := range groups[i].Rows {
			groups[i].Rows[j].Data = translateOutgoing(groups[i].Rows[j].Data, cm.idToCode)
		}
	}
	return groups, nil
}

func (s *RowService) ListRows(ctx context.Context, tableID int64, limit, offset int, filters []models.RowFilter, sorts []models.RowSort, search string) ([]models.Row, int64, error) {
	cm, columns, err := s.colMaps(ctx, tableID)
	if err != nil {
		return nil, 0, err
	}
	rows, total, err := s.Store.ListRows(ctx, tableID, limit, offset, filters, sorts, columns, search)
	if err != nil {
		return nil, 0, err
	}
	for i := range rows {
		rows[i].Data = translateOutgoing(rows[i].Data, cm.idToCode)
	}
	return rows, total, nil
}

func (s *RowService) CreateRow(ctx context.Context, tableID int64, data json.RawMessage) (*models.Row, error) {
	cm, columns, err := s.colMaps(ctx, tableID)
	if err != nil {
		return nil, err
	}
	row, err := s.Store.CreateRow(ctx, tableID, translateIncoming(data, cm.codeToID), columns)
	if err != nil {
		return nil, err
	}
	row.Data = translateOutgoing(row.Data, cm.idToCode)
	return row, nil
}

func (s *RowService) UpdateRow(ctx context.Context, tableID, rowID int64, data json.RawMessage, revisionID string) (*models.Row, string, error) {
	cm, columns, err := s.colMaps(ctx, tableID)
	if err != nil {
		return nil, "", err
	}
	row, rev, err := s.Store.UpdateRow(ctx, rowID, translateIncoming(data, cm.codeToID), revisionID, columns)
	if err != nil {
		return nil, "", err
	}
	row.Data = translateOutgoing(row.Data, cm.idToCode)
	return row, rev, nil
}

func (s *RowService) PatchRow(ctx context.Context, tableID, rowID int64, data json.RawMessage, revisionID string) (*models.Row, string, error) {
	cm, columns, err := s.colMaps(ctx, tableID)
	if err != nil {
		return nil, "", err
	}
	row, rev, err := s.Store.PatchRow(ctx, rowID, translateIncoming(data, cm.codeToID), revisionID, columns)
	if err != nil {
		return nil, "", err
	}
	row.Data = translateOutgoing(row.Data, cm.idToCode)
	return row, rev, nil
}

func (s *RowService) BulkPatch(ctx context.Context, tableID int64, filters []models.RowFilter, data json.RawMessage, preview bool) ([]*models.Row, int64, error) {
	cm, columns, err := s.colMaps(ctx, tableID)
	if err != nil {
		return nil, 0, err
	}
	rows, count, err := s.Store.BulkPatch(ctx, tableID, filters, translateIncoming(data, cm.codeToID), columns, preview)
	if err != nil {
		return nil, 0, err
	}
	for i := range rows {
		rows[i].Data = translateOutgoing(rows[i].Data, cm.idToCode)
	}
	return rows, count, nil
}

func (s *RowService) GetRow(ctx context.Context, tableID, rowID int64) (*models.Row, error) {
	cm, _, err := s.colMaps(ctx, tableID)
	if err != nil {
		return nil, err
	}
	row, err := s.Store.GetRow(ctx, rowID)
	if err != nil || row == nil {
		return row, err
	}
	row.Data = translateOutgoing(row.Data, cm.idToCode)
	return row, nil
}

func (s *RowService) ListRowHistory(ctx context.Context, tableID, rowID int64) ([]models.RowHistory, error) {
	cm, _, err := s.colMaps(ctx, tableID)
	if err != nil {
		return nil, err
	}
	entries, err := s.Store.ListRowHistory(ctx, rowID)
	if err != nil {
		return nil, err
	}
	for i := range entries {
		entries[i].Data = translateOutgoing(entries[i].Data, cm.idToCode)
	}
	return entries, nil
}

func (s *RowService) CreateAnnotation(ctx context.Context, rowID int64, text string) (*models.RowHistory, error) {
	return s.Store.CreateAnnotation(ctx, rowID, text)
}

func (s *RowService) UpdateAnnotation(ctx context.Context, historyID int64, text string) (*models.RowHistory, error) {
	return s.Store.UpdateAnnotation(ctx, historyID, text)
}

func (s *RowService) DeleteAnnotation(ctx context.Context, historyID int64) error {
	return s.Store.DeleteAnnotation(ctx, historyID)
}

func (s *RowService) UpdateChangeAnnotation(ctx context.Context, historyID int64, text string) (*models.RowHistory, error) {
	return s.Store.UpdateChangeAnnotation(ctx, historyID, text)
}
