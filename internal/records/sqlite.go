package records

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/saintedlama/degubase/internal/infrastructure/util"
	"github.com/saintedlama/degubase/internal/models"
)

// SQLite is the SQLite adapter for records.Store.
type SQLite struct {
	db *sql.DB
}

// NewSQLite creates a records.Store backed by SQLite.
func NewSQLite(db *sql.DB) *SQLite {
	return &SQLite{db: db}
}

var _ Store = (*SQLite)(nil)

// ── Rows ──────────────────────────────────────────────────────────────────────

func (s *SQLite) ListRows(ctx context.Context, tableID int64, limit, offset int, filters []models.RowFilter, sorts []models.RowSort, columns []models.Column, search string) ([]models.Row, int64, error) {
	if limit <= 0 {
		limit = 100
	}

	where, args := buildWhereClause(tableID, filters, sorts, columns, search)
	orderBy := buildOrderBy(sorts, columns)

	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM rows %s`, where)
	var total int64
	if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	querySQL := fmt.Sprintf(
		`SELECT id, table_id, data, created_at, updated_at FROM rows %s %s LIMIT ? OFFSET ?`,
		where, orderBy,
	)
	rows, err := s.db.QueryContext(ctx, querySQL, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.Row
	for rows.Next() {
		r, err := scanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *r)
	}
	return out, total, rows.Err()
}

func (s *SQLite) ListGroupedRows(ctx context.Context, tableID int64, groupByColCode string, limit int, filters []models.RowFilter, sorts []models.RowSort, columns []models.Column, search string) ([]models.RowGroup, error) {
	if limit <= 0 {
		limit = 200
	}

	groupCol := findColumnByCode(columns, groupByColCode)
	if groupCol == nil {
		return nil, fmt.Errorf("column %q not found", groupByColCode)
	}
	groupExpr := colExpr(groupCol)

	where, args := buildWhereClause(tableID, filters, sorts, columns, search)
	orderByContent := strings.TrimPrefix(buildOrderBy(sorts, columns), "ORDER BY ")

	querySQL := fmt.Sprintf(`
		WITH ranked AS (
			SELECT id, table_id, data, created_at, updated_at,
				%s AS group_val,
				ROW_NUMBER() OVER (PARTITION BY %s ORDER BY %s) AS rn,
				COUNT(*) OVER (PARTITION BY %s) AS grp_total
			FROM rows %s
		)
		SELECT id, table_id, data, created_at, updated_at, group_val, grp_total
		FROM ranked WHERE rn <= ?
		ORDER BY group_val NULLS LAST, rn`,
		groupExpr, groupExpr, orderByContent, groupExpr, where,
	)

	rows, err := s.db.QueryContext(ctx, querySQL, append(args, limit)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.RowGroup
	groupIndex := make(map[string]int)

	for rows.Next() {
		var r models.Row
		var dataStr, ca, ua string
		var groupVal sql.NullString
		var grpTotal int64
		if err := rows.Scan(&r.ID, &r.TableID, &dataStr, &ca, &ua, &groupVal, &grpTotal); err != nil {
			return nil, err
		}
		r.Data = json.RawMessage(dataStr)
		r.CreatedAt = util.ParseTime(ca)
		r.UpdatedAt = util.ParseTime(ua)

		// Use a sentinel key so NULL and empty-string groups don't collide.
		key := "\x00"
		if groupVal.Valid {
			key = groupVal.String
		}

		idx, exists := groupIndex[key]
		if !exists {
			g := models.RowGroup{Total: grpTotal}
			if groupVal.Valid {
				v := groupVal.String
				g.Value = &v
			}
			idx = len(result)
			result = append(result, g)
			groupIndex[key] = idx
		}
		result[idx].Rows = append(result[idx].Rows, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// colExpr returns the SQL expression to extract a column's value.
// Virtual columns (created_at, updated_at) map to real row columns.
// Regular columns use json_extract on the JSON data blob.
func colExpr(col *models.Column) string {
	switch col.Type {
	case models.ColumnTypeCreatedAt:
		return "rows.created_at"
	case models.ColumnTypeUpdatedAt:
		return "rows.updated_at"
	}
	return fmt.Sprintf("json_extract(data, '$.%d')", col.ID)
}

func findColumnByCode(columns []models.Column, code string) *models.Column {
	for i := range columns {
		if columns[i].Code == code {
			return &columns[i]
		}
	}
	return nil
}

var textSearchTypes = map[models.ColumnType]bool{
	models.ColumnTypeText:         true,
	models.ColumnTypeLongText:     true,
	models.ColumnTypeMarkdown:     true,
	models.ColumnTypeEmail:        true,
	models.ColumnTypeURL:          true,
	models.ColumnTypeSingleSelect: true,
	models.ColumnTypeMultiSelect:  true,
}

// buildWhereClause constructs the WHERE clause for ListRows.
func buildWhereClause(tableID int64, filters []models.RowFilter, sorts []models.RowSort, columns []models.Column, search string) (string, []any) {
	conds := []string{"table_id = ?"}
	args := []any{tableID}

	for _, f := range filters {
		col := findColumnByCode(columns, f.Col)
		if col == nil {
			continue
		}
		expr := colExpr(col)
		switch f.Op {
		case "is":
			conds = append(conds, fmt.Sprintf("%s = ?", expr))
			args = append(args, f.Value)
		case "is_not":
			conds = append(conds, fmt.Sprintf("(%s IS NULL OR %s != ?)", expr, expr))
			args = append(args, f.Value)
		case "contains":
			conds = append(conds, fmt.Sprintf("%s LIKE ? ESCAPE '\\'", expr))
			args = append(args, "%"+escapeLike(f.Value)+"%")
		case "not_contains":
			conds = append(conds, fmt.Sprintf("(%s IS NULL OR %s NOT LIKE ? ESCAPE '\\')", expr, expr))
			args = append(args, "%"+escapeLike(f.Value)+"%")
		case "is_empty":
			conds = append(conds, fmt.Sprintf("(%s IS NULL OR %s = '')", expr, expr))
		case "is_not_empty":
			conds = append(conds, fmt.Sprintf("(%s IS NOT NULL AND %s != '')", expr, expr))
		case "eq":
			conds = append(conds, fmt.Sprintf("CAST(%s AS REAL) = CAST(? AS REAL)", expr))
			args = append(args, f.Value)
		case "gt":
			conds = append(conds, fmt.Sprintf("CAST(%s AS REAL) > CAST(? AS REAL)", expr))
			args = append(args, f.Value)
		case "lt":
			conds = append(conds, fmt.Sprintf("CAST(%s AS REAL) < CAST(? AS REAL)", expr))
			args = append(args, f.Value)
		case "before":
			conds = append(conds, fmt.Sprintf("%s < ?", expr))
			args = append(args, f.Value)
		case "after":
			conds = append(conds, fmt.Sprintf("%s > ?", expr))
			args = append(args, f.Value)
		case "is_true":
			conds = append(conds, fmt.Sprintf("(%s = 1 OR %s = 'true')", expr, expr))
		case "is_false":
			conds = append(conds, fmt.Sprintf("(%s = 0 OR %s = 'false' OR %s IS NULL)", expr, expr, expr))
		case "in":
			var targets []string
			if json.Unmarshal([]byte(f.Value), &targets) == nil && len(targets) > 0 {
				placeholders := strings.Repeat("?,", len(targets))
				placeholders = placeholders[:len(placeholders)-1]
				conds = append(conds, fmt.Sprintf("%s IN (%s)", expr, placeholders))
				for _, t := range targets {
					args = append(args, t)
				}
			}
		case "not_in":
			var targets []string
			if json.Unmarshal([]byte(f.Value), &targets) == nil && len(targets) > 0 {
				placeholders := strings.Repeat("?,", len(targets))
				placeholders = placeholders[:len(placeholders)-1]
				conds = append(conds, fmt.Sprintf("(%s IS NULL OR %s NOT IN (%s))", expr, expr, placeholders))
				for _, t := range targets {
					args = append(args, t)
				}
			}
		}
	}

	// Full-text search across text-type columns
	if search != "" {
		var searchConds []string
		pattern := "%" + escapeLike(search) + "%"
		for _, col := range columns {
			if !textSearchTypes[col.Type] {
				continue
			}
			searchConds = append(searchConds, fmt.Sprintf("%s LIKE ? ESCAPE '\\'", colExpr(&col)))
			args = append(args, pattern)
		}
		if len(searchConds) > 0 {
			conds = append(conds, "("+strings.Join(searchConds, " OR ")+")")
		}
	}

	return "WHERE " + strings.Join(conds, " AND "), args
}

// buildOrderBy constructs the ORDER BY clause for ListRows.
func buildOrderBy(sorts []models.RowSort, columns []models.Column) string {
	if len(sorts) == 0 {
		return "ORDER BY rows.id"
	}
	var parts []string
	for _, s := range sorts {
		col := findColumnByCode(columns, s.Col)
		if col == nil {
			continue
		}
		dir := "ASC"
		if strings.ToLower(s.Dir) == "desc" {
			dir = "DESC"
		}
		parts = append(parts, fmt.Sprintf("%s %s", colExpr(col), dir))
	}
	if len(parts) == 0 {
		return "ORDER BY rows.id"
	}
	// Always append id as a tiebreaker for stable pagination
	parts = append(parts, "rows.id ASC")
	return "ORDER BY " + strings.Join(parts, ", ")
}

// escapeLike escapes LIKE pattern special characters.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// ── row_links helpers ─────────────────────────────────────────────────────────

// extractLinkEntries scans ID-keyed row data and returns one RowLinkEntry per
// non-null row-link cell.
func extractLinkEntries(data json.RawMessage, columns []models.Column) []models.RowLinkEntry {
	var dataMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &dataMap); err != nil {
		return nil
	}
	var out []models.RowLinkEntry
	for _, col := range columns {
		if col.Type != models.ColumnTypeRowLink {
			continue
		}
		raw, ok := dataMap[strconv.FormatInt(col.ID, 10)]
		if !ok || string(raw) == "null" {
			continue
		}
		var cell struct {
			ID int64 `json:"id"`
		}
		if json.Unmarshal(raw, &cell) == nil && cell.ID > 0 {
			out = append(out, models.RowLinkEntry{ColID: col.ID, TargetRowID: cell.ID})
		}
	}
	return out
}

// syncRowLinks replaces all row_links entries for sourceRowID with the provided
// entries inside the given transaction. Target rows that no longer exist are
// silently skipped (orphaned cell values keep no entry in row_links).
// target_col_id is resolved via subquery from the source column's options
// (displayColumnCode) or the target table's first text column.
func syncRowLinks(ctx context.Context, tx *sql.Tx, sourceRowID, sourceTableID int64, entries []models.RowLinkEntry) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM row_links WHERE source_row_id = ?`, sourceRowID); err != nil {
		return err
	}
	for _, e := range entries {
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO row_links
			 (source_row_id, source_col_id, source_table_id, target_row_id, target_table_id, target_col_id)
			 SELECT ?, ?, ?, r.id, r.table_id,
			  COALESCE(
			   (SELECT tc.id FROM columns tc
			    WHERE tc.table_id = r.table_id
			    AND tc.code = (SELECT json_extract(sc.options, '$.displayColumnCode')
			                   FROM columns sc WHERE sc.id = ?)
			   ),
			   (SELECT tc.id FROM columns tc
			    WHERE tc.table_id = r.table_id
			    AND tc.type IN ('text','long-text','markdown','email','url')
			    ORDER BY tc.position LIMIT 1)
			  )
			 FROM rows r WHERE r.id = ?`,
			sourceRowID, e.ColID, sourceTableID, e.ColID, e.TargetRowID,
		); err != nil {
			return err
		}
	}
	return nil
}

// BulkPatch merges patch into every row matching filters in one transaction.
// preview=true counts matches without writing.
func (s *SQLite) BulkPatch(ctx context.Context, tableID int64, filters []models.RowFilter, patch json.RawMessage, columns []models.Column, preview bool) ([]*models.Row, int64, error) {
	where, args := buildWhereClause(tableID, filters, nil, columns, "")
	if preview {
		var count int64
		err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM rows "+where, args...).Scan(&count)
		return nil, count, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, "SELECT id, data FROM rows "+where, args...)
	if err != nil {
		return nil, 0, err
	}
	type rowSnapshot struct {
		id   int64
		data string
	}
	var snapshots []rowSnapshot
	for rows.Next() {
		var snap rowSnapshot
		if err := rows.Scan(&snap.id, &snap.data); err != nil {
			rows.Close()
			return nil, 0, err
		}
		snapshots = append(snapshots, snap)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	var updatedIDs []int64
	for _, snap := range snapshots {
		merged, err := mergeJSON(json.RawMessage(snap.data), patch)
		if err != nil {
			return nil, 0, err
		}
		revID := uuid.New().String()
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO row_history (row_id, data, changed_at, revision_id) VALUES (?, ?, ?, ?)`,
			snap.id, snap.data, now, revID); err != nil {
			return nil, 0, err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE rows SET data = ?, updated_at = ? WHERE id = ?`,
			string(merged), now, snap.id); err != nil {
			return nil, 0, err
		}
		if err := syncRowLinks(ctx, tx, snap.id, tableID, extractLinkEntries(merged, columns)); err != nil {
			return nil, 0, err
		}
		updatedIDs = append(updatedIDs, snap.id)
	}

	if err := tx.Commit(); err != nil {
		return nil, 0, err
	}

	var updated []*models.Row
	for _, id := range updatedIDs {
		row, err := s.GetRow(ctx, id)
		if err != nil || row == nil {
			continue
		}
		updated = append(updated, row)
	}
	return updated, int64(len(updatedIDs)), nil
}

func (s *SQLite) CreateRow(ctx context.Context, tableID int64, data json.RawMessage, columns []models.Column) (*models.Row, error) {
	if len(data) == 0 {
		data = json.RawMessage("{}")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx,
		`INSERT INTO rows (table_id, data) VALUES (?, ?)`, tableID, string(data))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	_, err = tx.ExecContext(ctx,
		`INSERT INTO row_history (row_id, data, changed_at, revision_id) VALUES (?, ?, ?, ?)`,
		id, string(data), now, uuid.New().String())
	if err != nil {
		return nil, err
	}
	if err := syncRowLinks(ctx, tx, id, tableID, extractLinkEntries(data, columns)); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetRow(ctx, id)
}

func (s *SQLite) UpdateRow(ctx context.Context, id int64, data json.RawMessage, revisionID string, columns []models.Column) (*models.Row, string, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback()
	var oldData string
	var tableID int64
	if err := tx.QueryRowContext(ctx, `SELECT data, table_id FROM rows WHERE id = ?`, id).Scan(&oldData, &tableID); err != nil {
		return nil, "", err
	}
	revisionID, err = upsertRevision(ctx, tx, id, oldData, now, revisionID)
	if err != nil {
		return nil, "", err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE rows SET data = ?, updated_at = ? WHERE id = ?`,
		string(data), now, id); err != nil {
		return nil, "", err
	}
	if err := syncRowLinks(ctx, tx, id, tableID, extractLinkEntries(data, columns)); err != nil {
		return nil, "", err
	}
	if err := tx.Commit(); err != nil {
		return nil, "", err
	}
	row, err := s.GetRow(ctx, id)
	return row, revisionID, err
}

func (s *SQLite) PatchRow(ctx context.Context, id int64, patch json.RawMessage, revisionID string, columns []models.Column) (*models.Row, string, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback()
	var oldData string
	var tableID int64
	if err := tx.QueryRowContext(ctx, `SELECT data, table_id FROM rows WHERE id = ?`, id).Scan(&oldData, &tableID); err != nil {
		return nil, "", err
	}
	merged, err := mergeJSON(json.RawMessage(oldData), patch)
	if err != nil {
		return nil, "", err
	}
	revisionID, err = upsertRevision(ctx, tx, id, oldData, now, revisionID)
	if err != nil {
		return nil, "", err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE rows SET data = ?, updated_at = ? WHERE id = ?`,
		string(merged), now, id); err != nil {
		return nil, "", err
	}
	if err := syncRowLinks(ctx, tx, id, tableID, extractLinkEntries(merged, columns)); err != nil {
		return nil, "", err
	}
	if err := tx.Commit(); err != nil {
		return nil, "", err
	}
	row, err := s.GetRow(ctx, id)
	return row, revisionID, err
}

// upsertRevision either bundles a save into an existing revision (updating only
// the timestamp so the stored "before" snapshot stays intact) or creates a new
// history entry. Returns the revision ID that was used.
func upsertRevision(ctx context.Context, tx *sql.Tx, rowID int64, oldData, now, revisionID string) (string, error) {
	if revisionID != "" {
		res, err := tx.ExecContext(ctx,
			`UPDATE row_history SET changed_at = ? WHERE revision_id = ? AND row_id = ?`,
			now, revisionID, rowID)
		if err != nil {
			return "", err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			return revisionID, nil
		}
		// Unknown revision — fall through and create a new history entry with a fresh ID.
	}
	revisionID = uuid.New().String()
	_, err := tx.ExecContext(ctx,
		`INSERT INTO row_history (row_id, data, changed_at, revision_id) VALUES (?, ?, ?, ?)`,
		rowID, oldData, now, revisionID)
	return revisionID, err
}

func (s *SQLite) ListRowHistory(ctx context.Context, rowID int64) ([]models.RowHistory, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, row_id, COALESCE(entry_type,'change'), COALESCE(data,''), changed_at, COALESCE(revision_id,''), COALESCE(annotation,'')
		 FROM row_history WHERE row_id = ? ORDER BY id DESC`,
		rowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.RowHistory
	for rows.Next() {
		var h models.RowHistory
		var dataStr, ca string
		if err := rows.Scan(&h.ID, &h.RowID, &h.EntryType, &dataStr, &ca, &h.RevisionID, &h.Annotation); err != nil {
			return nil, err
		}
		if dataStr != "" && h.EntryType == "change" {
			h.Data = json.RawMessage(dataStr)
		}
		h.ChangedAt = util.ParseTime(ca)
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *SQLite) CreateAnnotation(ctx context.Context, rowID int64, text string) (*models.RowHistory, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO row_history (row_id, entry_type, annotation, changed_at) VALUES (?, 'annotation', ?, ?)`,
		rowID, text, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &models.RowHistory{
		ID:         id,
		RowID:      rowID,
		EntryType:  "annotation",
		Annotation: text,
		ChangedAt:  util.ParseTime(now),
	}, nil
}

func (s *SQLite) UpdateAnnotation(ctx context.Context, rowID, historyID int64, text string) (*models.RowHistory, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE row_history SET annotation = ? WHERE id = ? AND row_id = ? AND entry_type = 'annotation'`,
		text, historyID, rowID)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, fmt.Errorf("annotation not found")
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT id, row_id, COALESCE(entry_type,'change'), COALESCE(data,''), changed_at, COALESCE(revision_id,''), COALESCE(annotation,'')
		 FROM row_history WHERE id = ?`, historyID)
	var h models.RowHistory
	var dataStr, ca string
	if err := row.Scan(&h.ID, &h.RowID, &h.EntryType, &dataStr, &ca, &h.RevisionID, &h.Annotation); err != nil {
		return nil, err
	}
	h.ChangedAt = util.ParseTime(ca)
	return &h, nil
}

func (s *SQLite) DeleteAnnotation(ctx context.Context, rowID, historyID int64) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM row_history WHERE id = ? AND row_id = ? AND entry_type = 'annotation'`,
		historyID, rowID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("annotation not found")
	}
	return nil
}

func (s *SQLite) UpdateChangeAnnotation(ctx context.Context, rowID, historyID int64, text string) (*models.RowHistory, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE row_history SET annotation = ? WHERE id = ? AND row_id = ? AND entry_type = 'change'`,
		text, historyID, rowID)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, fmt.Errorf("history entry not found")
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT id, row_id, COALESCE(entry_type,'change'), COALESCE(data,''), changed_at, COALESCE(revision_id,''), COALESCE(annotation,'')
		 FROM row_history WHERE id = ?`, historyID)
	var h models.RowHistory
	var dataStr, ca string
	if err := row.Scan(&h.ID, &h.RowID, &h.EntryType, &dataStr, &ca, &h.RevisionID, &h.Annotation); err != nil {
		return nil, err
	}
	if dataStr != "" {
		h.Data = json.RawMessage(dataStr)
	}
	h.ChangedAt = util.ParseTime(ca)
	return &h, nil
}

func (s *SQLite) GetRow(ctx context.Context, id int64) (*models.Row, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, table_id, data, created_at, updated_at FROM rows WHERE id = ?`, id)
	r, err := scanRowRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

func (s *SQLite) DeleteRow(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM rows WHERE id = ?`, id)
	return err
}

func (s *SQLite) FindRowLinkReferences(ctx context.Context, workspaceID, targetRowID int64) ([]models.RowLinkRef, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT rl.source_row_id, src_t.code, src_t.name
		FROM row_links rl
		JOIN tables src_t  ON src_t.id  = rl.source_table_id
		JOIN rows   src_r  ON src_r.id  = rl.source_row_id
		JOIN columns c     ON c.id      = rl.source_col_id
		WHERE rl.target_row_id = ?
		  AND src_t.workspace_id = ?
		LIMIT 20 -- arbitrary safety cap; the dialog shows all of them
	`, targetRowID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var refs []models.RowLinkRef
	for rows.Next() {
		var r models.RowLinkRef
		if err := rows.Scan(&r.RowID, &r.TableCode, &r.TableName); err != nil {
			return nil, err
		}
		refs = append(refs, r)
	}
	return refs, rows.Err()
}

func (s *SQLite) ListLinksForRows(ctx context.Context, sourceRowIDs []int64) ([]RowLinkInfo, error) {
	if len(sourceRowIDs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(sourceRowIDs))
	args := make([]interface{}, len(sourceRowIDs))
	for i, id := range sourceRowIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(`
		SELECT rl.source_row_id, sc.code, rl.target_row_id, rl.target_table_id, tc.code
		FROM row_links rl
		JOIN columns sc ON sc.id = rl.source_col_id
		LEFT JOIN columns tc ON tc.id = rl.target_col_id
		WHERE rl.source_row_id IN (%s)`, strings.Join(placeholders, ","))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RowLinkInfo
	for rows.Next() {
		var li RowLinkInfo
		var targetColCode sql.NullString
		if err := rows.Scan(&li.SourceRowID, &li.SourceColCode, &li.TargetRowID, &li.TargetTableID, &targetColCode); err != nil {
			return nil, err
		}
		li.TargetColCode = targetColCode.String
		out = append(out, li)
	}
	return out, rows.Err()
}

func (s *SQLite) ListReferencingRows(ctx context.Context, workspaceID, targetRowID int64, limit, offset int) ([]ReferencingRowRef, int64, error) {
	if limit <= 0 {
		limit = 50
	}

	var total int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*)
		 FROM row_links rl
		 JOIN tables src_t ON src_t.id = rl.source_table_id
		 WHERE rl.target_row_id = ? AND src_t.workspace_id = ?`,
		targetRowID, workspaceID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT rl.source_row_id, src_t.id, src_t.code, src_t.name, c.code, c.name, src_r.data
		 FROM row_links rl
		 JOIN tables src_t ON src_t.id = rl.source_table_id
		 JOIN columns c    ON c.id    = rl.source_col_id
		 JOIN rows src_r   ON src_r.id = rl.source_row_id
		 WHERE rl.target_row_id = ? AND src_t.workspace_id = ?
		 ORDER BY src_t.code, c.code
		 LIMIT ? OFFSET ?`,
		targetRowID, workspaceID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []ReferencingRowRef
	for rows.Next() {
		var r ReferencingRowRef
		if err := rows.Scan(&r.SourceRowID, &r.SourceTableID, &r.SourceTableCode, &r.SourceTableName, &r.ViaColumnCode, &r.ViaColumnName, &r.SourceRowData); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}

// ── Scan helpers (local) ──────────────────────────────────────────────────────

func scanRow(rs interface{ Scan(...any) error }) (*models.Row, error) {
	var r models.Row
	var dataStr, ca, ua string
	if err := rs.Scan(&r.ID, &r.TableID, &dataStr, &ca, &ua); err != nil {
		return nil, err
	}
	r.Data = json.RawMessage(dataStr)
	r.CreatedAt = util.ParseTime(ca)
	r.UpdatedAt = util.ParseTime(ua)
	return &r, nil
}

func scanRowRow(r *sql.Row) (*models.Row, error) {
	var row models.Row
	var dataStr, ca, ua string
	if err := r.Scan(&row.ID, &row.TableID, &dataStr, &ca, &ua); err != nil {
		return nil, err
	}
	row.Data = json.RawMessage(dataStr)
	row.CreatedAt = util.ParseTime(ca)
	row.UpdatedAt = util.ParseTime(ua)
	return &row, nil
}
