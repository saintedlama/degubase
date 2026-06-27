package records

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/models"
	"github.com/saintedlama/degubase/internal/schema"
)

// ReferencingHandler lists records that reference a given row.
type ReferencingHandler struct {
	Store Store
	Cols  ColumnLister
	Svc   *RowService
	Tbls  schema.TableStore
}

// @Summary     List referencing records
// @Description Returns paginated records from any table in the workspace that
// @Description link to the given row via a row-link column.
// @Tags        rows
// @Produce     json
// @Param       wsCode     path      string  true   "workspace code"
// @Param       tableCode  path      string  true   "table code"
// @Param       rowID      path      int     true   "row ID"
// @Param       page       query     int     false  "page"
// @Param       pageSize   query     int     false  "page size"
// @Success     200        {object}  models.PagedReferencingRows
// @Failure     404        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows/{rowID}/referencing [get]
func (h *ReferencingHandler) ListReferencing(w http.ResponseWriter, r *http.Request) {
	rowID, err := strconv.ParseInt(chi.URLParam(r, "rowID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid row id")
		return
	}

	ws := httplib.WorkspaceFromCtx(r)
	t := httplib.TableFromCtx(r)
	paging := httplib.ParsePaging(r)

	// Verify the target row exists.
	row, err := h.Svc.GetRow(r.Context(), t.ID, rowID)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if row == nil {
		httplib.NotFound(w)
		return
	}

	refs, total, err := h.Store.ListReferencingRows(r.Context(), ws.ID, rowID, paging.PageSize, paging.Offset())
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}

	// Resolve source row labels.
	out := make([]models.ReferencingRow, len(refs))
	for i, ref := range refs {
		out[i] = models.ReferencingRow{
			SourceRowID:     ref.SourceRowID,
			SourceTableCode: ref.SourceTableCode,
			SourceTableName: ref.SourceTableName,
			SourceRowLabel:  resolveReferencingLabel(r.Context(), h.Cols, ref),
			ViaColumnCode:   ref.ViaColumnCode,
			ViaColumnName:   ref.ViaColumnName,
		}
	}

	result := models.NewPagedResult(out, total, paging.Page, paging.PageSize)
	httplib.Respond(w, http.StatusOK, result)
}

// resolveReferencingLabel extracts a human-readable label from a source row's
// data. The data is ID-keyed (raw from the DB). Uses the first text column on
// the source table by position, or "Row #N" as fallback.
func resolveReferencingLabel(ctx context.Context, cols ColumnLister, ref ReferencingRowRef) string {
	if ref.SourceRowData == "" || ref.SourceRowData == "{}" {
		return fmt.Sprintf("Row #%d", ref.SourceRowID)
	}

	tblCols, err := cols.ListColumns(ctx, ref.SourceTableID)
	if err != nil || len(tblCols) == 0 {
		return fmt.Sprintf("Row #%d", ref.SourceRowID)
	}

	var dataMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(ref.SourceRowData), &dataMap); err != nil || len(dataMap) == 0 {
		return fmt.Sprintf("Row #%d", ref.SourceRowID)
	}

	// Find the first text column by position.
	var firstTextColID int64
	firstTextPos := -1
	for _, c := range tblCols {
		if c.ID <= 0 {
			continue
		}
		switch c.Type {
		case models.ColumnTypeText, models.ColumnTypeLongText, models.ColumnTypeMarkdown,
			models.ColumnTypeEmail, models.ColumnTypeURL:
			if firstTextPos < 0 || c.Position < firstTextPos {
				firstTextColID = c.ID
				firstTextPos = c.Position
			}
		}
	}

	// Try the first text column by ID.
	if firstTextColID > 0 {
		key := strconv.FormatInt(firstTextColID, 10)
		if raw, ok := dataMap[key]; ok {
			var s string
			if json.Unmarshal(raw, &s) == nil && s != "" {
				return s
			}
		}
	}

	// Fall back to any text column.
	for _, c := range tblCols {
		if c.ID <= 0 {
			continue
		}
		switch c.Type {
		case models.ColumnTypeText, models.ColumnTypeLongText, models.ColumnTypeMarkdown,
			models.ColumnTypeEmail, models.ColumnTypeURL:
			key := strconv.FormatInt(c.ID, 10)
			if raw, ok := dataMap[key]; ok {
				var s string
				if json.Unmarshal(raw, &s) == nil && s != "" {
					return s
				}
			}
		}
	}

	return fmt.Sprintf("Row #%d", ref.SourceRowID)
}
