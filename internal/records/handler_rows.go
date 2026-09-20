package records

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/saintedlama/degubase/internal/infrastructure/events"
	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/models"
	"github.com/saintedlama/degubase/internal/schema"
)

// colMaps holds bidirectional ID↔code maps for a table's columns.
type colMaps struct {
	idToCode map[string]string // "11" → "title"
	codeToID map[string]string // "title" → "11"
}

func buildColMaps(columns []models.Column) colMaps {
	m := colMaps{
		idToCode: make(map[string]string, len(columns)),
		codeToID: make(map[string]string, len(columns)),
	}
	for _, c := range columns {
		if c.ID < 0 || c.Code == "" {
			continue // virtual columns have no data keys
		}
		key := strconv.FormatInt(c.ID, 10)
		m.idToCode[key] = c.Code
		m.codeToID[c.Code] = key
	}
	return m
}

// translateIncoming rewrites code keys to numeric-ID string keys for DB storage.
// Keys not in codeToID are silently dropped.
func translateIncoming(data json.RawMessage, codeToID map[string]string) json.RawMessage {
	if len(data) == 0 {
		return data
	}
	var src map[string]json.RawMessage
	if err := json.Unmarshal(data, &src); err != nil {
		return data
	}
	dst := make(map[string]json.RawMessage, len(src))
	for k, v := range src {
		if idKey, ok := codeToID[k]; ok {
			dst[idKey] = v
		}
		// keys not in codeToID (stale IDs, virtual col codes, unknowns) are dropped
	}
	out, err := json.Marshal(dst)
	if err != nil {
		return data
	}
	return out
}

// applyDataSelect filters a code-keyed data object to only the requested columns.
// A nil/empty cols slice is a no-op (all columns kept).
func applyDataSelect(data json.RawMessage, cols []string) json.RawMessage {
	if len(cols) == 0 || len(data) == 0 {
		return data
	}
	var src map[string]json.RawMessage
	if err := json.Unmarshal(data, &src); err != nil {
		return data
	}
	dst := make(map[string]json.RawMessage, len(cols))
	for _, c := range cols {
		if v, ok := src[c]; ok {
			dst[c] = v
		}
	}
	out, err := json.Marshal(dst)
	if err != nil {
		return data
	}
	return out
}

// translateOutgoing rewrites numeric-ID string keys to code keys for API responses.
// Keys not in idToCode are silently dropped (removes stale/garbage entries).
func translateOutgoing(data json.RawMessage, idToCode map[string]string) json.RawMessage {
	if len(data) == 0 {
		return data
	}
	var src map[string]json.RawMessage
	if err := json.Unmarshal(data, &src); err != nil {
		return data
	}
	dst := make(map[string]json.RawMessage, len(src))
	for k, v := range src {
		if colCode, ok := idToCode[k]; ok {
			dst[colCode] = v
		}
		// keys not in idToCode (stale virtual col IDs, deleted cols, etc.) are dropped
	}
	out, err := json.Marshal(dst)
	if err != nil {
		return data
	}
	return out
}

type RowHandler struct {
	Store  Store
	Cols   ColumnLister
	Svc    *RowService
	Broker *events.Broker
	Files  *FileHandler
	// Tbls is optional. When set, row-link cell labels are refreshed from the
	// referenced row on every read so stale denormalized labels are never served.
	Tbls schema.TableStore
}

// resolveRowLinkLabels patches the label inside every row-link cell value
// ({id, label}) with the referenced row's current first-text-column value.
// Orphaned references (deleted rows, deleted tables) keep their stored label
// as a readable fallback. Results are cached per call to avoid N+1 queries.
func (h *RowHandler) resolveRowLinkLabels(ctx context.Context, wsID int64, rows []models.Row, cols []models.Column) {
	if h.Tbls == nil || len(rows) == 0 {
		return
	}

	// Quick check: any row-link columns at all?
	hasRowLink := false
	for _, col := range cols {
		if col.Type == models.ColumnTypeRowLink {
			hasRowLink = true
			break
		}
	}
	if !hasRowLink {
		return
	}

	// Collect source row IDs.
	sourceIDs := make([]int64, len(rows))
	for i, row := range rows {
		sourceIDs[i] = row.ID
	}

	// Fetch resolved link info from row_links (includes pre-resolved target_col_id).
	links, err := h.Store.ListLinksForRows(ctx, sourceIDs)
	if err != nil || len(links) == 0 {
		return
	}

	// Group by source row ID.
	linksBySource := make(map[int64][]RowLinkInfo, len(sourceIDs))
	for _, li := range links {
		linksBySource[li.SourceRowID] = append(linksBySource[li.SourceRowID], li)
	}

	// Per-request caches.
	labelCache := make(map[string]string) // "<tableID>:<rowID>" → label

	for i := range rows {
		rowLinks := linksBySource[rows[i].ID]
		if len(rowLinks) == 0 {
			continue
		}

		var dataMap map[string]json.RawMessage
		if err := json.Unmarshal(rows[i].Data, &dataMap); err != nil {
			continue
		}
		changed := false

		for _, li := range rowLinks {
			raw, ok := dataMap[li.SourceColCode]
			if !ok || string(raw) == "null" {
				continue
			}
			var link struct {
				ID    int64  `json:"id"`
				Label string `json:"label"`
			}
			if err := json.Unmarshal(raw, &link); err != nil || link.ID == 0 {
				continue
			}

			cacheKey := strconv.FormatInt(li.TargetTableID, 10) + ":" + strconv.FormatInt(link.ID, 10)
			freshLabel, hit := labelCache[cacheKey]
			if !hit {
				ref, err := h.Svc.GetRow(ctx, li.TargetTableID, link.ID)
				if err != nil || ref == nil || ref.TableID != li.TargetTableID {
					// Orphaned — keep stored label; cache it so we don't retry.
					labelCache[cacheKey] = link.Label
					continue
				}
				if li.TargetColCode != "" {
					freshLabel = rowLabelByCode(ref.Data, li.TargetColCode)
				}
				if freshLabel == "" {
					// Fallback: resolve display column the old way (legacy data
					// without target_col_id, or target column deletion).
					freshLabel = h.resolveLegacyLabel(ctx, ref, li.SourceColCode, cols)
				}
				labelCache[cacheKey] = freshLabel
			}

			if freshLabel == link.Label {
				continue
			}
			link.Label = freshLabel
			if newRaw, err := json.Marshal(link); err == nil {
				dataMap[li.SourceColCode] = newRaw
				changed = true
			}
		}

		if changed {
			if newData, err := json.Marshal(dataMap); err == nil {
				rows[i].Data = newData
			}
		}
	}
}

// resolveLegacyLabel resolves a link label the old way (parsing column options,
// listing target table columns). Only used as a fallback for links whose
// target_col_id was not resolved at write time (pre-migration data).
func (h *RowHandler) resolveLegacyLabel(ctx context.Context, ref *models.Row, sourceColCode string, cols []models.Column) string {
	var sourceCol models.Column
	found := false
	for _, col := range cols {
		if col.Code == sourceColCode {
			sourceCol = col
			found = true
			break
		}
	}
	if !found || sourceCol.Options == nil {
		return graphRowLabel(ref, nil)
	}
	var opts struct {
		DisplayColumnCode string `json:"displayColumnCode"`
	}
	if err := json.Unmarshal(sourceCol.Options, &opts); err != nil {
		return graphRowLabel(ref, nil)
	}
	if opts.DisplayColumnCode != "" {
		if label := rowLabelByCode(ref.Data, opts.DisplayColumnCode); label != "" {
			return label
		}
	}
	// List target columns for first-text-column fallback.
	tblCols, err := h.Cols.ListColumns(ctx, ref.TableID)
	if err != nil {
		return ""
	}
	return graphRowLabel(ref, tblCols)
}

// @Summary     List rows grouped by a column value
// @Tags        rows
// @Produce     json
// @Param       wsCode     path      string  true   "workspace code"
// @Param       tableCode  path      string  true   "table code"
// @Param       groupBy    query     string  true   "column code to group by"
// @Param       pageSize   query     int     false  "max rows per group (default 200)"
// @Param       filter     query     string  false  "shorthand filter: col:value or col:op:value; repeat for AND; pipe-separate values for in/not_in"
// @Param       filters    query     string  false  "JSON-encoded filters (alternative to filter)"
// @Param       sort       query     string  false  "col:dir,col:dir  or  JSON-encoded sort array"
// @Param       q          query     string  false  "search query"
// @Success     200        {array}   models.RowGroup
// @Failure     400        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/groups [get]
func (h *RowHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	t := httplib.TableFromCtx(r)

	groupBy := r.URL.Query().Get("groupBy")
	if groupBy == "" {
		httplib.BadRequest(w, "groupBy is required")
		return
	}

	paging := httplib.ParsePaging(r)

	var filters []models.RowFilter
	if shorthands := r.URL.Query()["filter"]; len(shorthands) > 0 {
		parsed, err := parseFilterShorthands(shorthands)
		if err != nil {
			httplib.BadRequest(w, err.Error())
			return
		}
		filters = append(filters, parsed...)
	}
	if raw := r.URL.Query().Get("filters"); raw != "" {
		var jsonFilters []models.RowFilter
		if err := json.Unmarshal([]byte(raw), &jsonFilters); err != nil {
			httplib.BadRequest(w, "invalid filters")
			return
		}
		filters = append(filters, jsonFilters...)
	}

	var rowSorts []models.RowSort
	if raw := r.URL.Query().Get("sort"); raw != "" {
		if strings.HasPrefix(raw, "[") {
			if err := json.Unmarshal([]byte(raw), &rowSorts); err != nil {
				var s models.RowSort
				if err2 := json.Unmarshal([]byte(raw), &s); err2 != nil {
					httplib.BadRequest(w, "invalid sort")
					return
				}
				if s.Col != "" {
					rowSorts = []models.RowSort{s}
				}
			}
		} else {
			parsed, err := parseSortShorthand(raw)
			if err != nil {
				httplib.BadRequest(w, err.Error())
				return
			}
			rowSorts = parsed
		}
		valid := rowSorts[:0:0]
		for _, s := range rowSorts {
			if s.Col != "" {
				valid = append(valid, s)
			}
		}
		rowSorts = valid
	}

	search := r.URL.Query().Get("q")

	groups, err := h.Svc.ListGroupedRows(r.Context(), t.ID, groupBy, paging.PageSize, filters, rowSorts, search)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if groups == nil {
		groups = []models.RowGroup{}
	}

	if ws := httplib.WorkspaceFromCtx(r); ws != nil {
		cols, _ := h.Cols.ListColumns(r.Context(), t.ID)
		for i := range groups {
			h.resolveRowLinkLabels(r.Context(), ws.ID, groups[i].Rows, cols)
		}
	}

	httplib.Respond(w, http.StatusOK, groups)
}

// @Summary     List rows
// @Tags        rows
// @Produce     json
// @Param       wsCode     path      string  true   "workspace code"
// @Param       tableCode  path      string  true   "table code"
// @Param       page       query     int     false  "page"
// @Param       pageSize   query     int     false  "page size"
// @Param       filter     query     string  false  "shorthand filter: col:value or col:op:value; repeat for AND; pipe-separate values for in/not_in"
// @Param       filters    query     string  false  "JSON-encoded filters (alternative to filter)"
// @Param       sort       query     string  false  "col:dir,col:dir  or  JSON-encoded sort array"
// @Param       q          query     string  false  "search query"
// @Param       select     query     string  false  "comma-separated column codes to include in data; omit to return all columns"
// @Success     200        {object}  models.PagedRows
// @Failure     400        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows [get]
func (h *RowHandler) List(w http.ResponseWriter, r *http.Request) {
	t := httplib.TableFromCtx(r)
	paging := httplib.ParsePaging(r)

	var filters []models.RowFilter

	// Shorthand: repeated filter=col:op:value params
	if shorthands := r.URL.Query()["filter"]; len(shorthands) > 0 {
		parsed, err := parseFilterShorthands(shorthands)
		if err != nil {
			httplib.BadRequest(w, err.Error())
			return
		}
		filters = append(filters, parsed...)
	}

	// JSON: filters=[{"col":"status","op":"is","value":"Open"}]
	if raw := r.URL.Query().Get("filters"); raw != "" {
		var jsonFilters []models.RowFilter
		if err := json.Unmarshal([]byte(raw), &jsonFilters); err != nil {
			httplib.BadRequest(w, "invalid filters")
			return
		}
		filters = append(filters, jsonFilters...)
	}

	var rowSorts []models.RowSort
	if raw := r.URL.Query().Get("sort"); raw != "" {
		if strings.HasPrefix(raw, "[") {
			// JSON array: sort=[{"col":"created_at","dir":"asc"}]
			if err := json.Unmarshal([]byte(raw), &rowSorts); err != nil {
				var s models.RowSort
				if err2 := json.Unmarshal([]byte(raw), &s); err2 != nil {
					httplib.BadRequest(w, "invalid sort")
					return
				}
				if s.Col != "" {
					rowSorts = []models.RowSort{s}
				}
			}
		} else {
			// Shorthand: sort=col:dir,col:dir
			parsed, err := parseSortShorthand(raw)
			if err != nil {
				httplib.BadRequest(w, err.Error())
				return
			}
			rowSorts = parsed
		}
		valid := rowSorts[:0:0]
		for _, s := range rowSorts {
			if s.Col != "" {
				valid = append(valid, s)
			}
		}
		rowSorts = valid
	}

	search := r.URL.Query().Get("q")

	var selectedCols []string
	if sel := r.URL.Query().Get("select"); sel != "" {
		for s := range strings.SplitSeq(sel, ",") {
			if s = strings.TrimSpace(s); s != "" {
				selectedCols = append(selectedCols, s)
			}
		}
	}

	rows, total, err := h.Svc.ListRows(r.Context(), t.ID, paging.PageSize, paging.Offset(), filters, rowSorts, search)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if rows == nil {
		rows = []models.Row{}
	}

	if ws := httplib.WorkspaceFromCtx(r); ws != nil {
		cols, _ := h.Cols.ListColumns(r.Context(), t.ID)
		h.resolveRowLinkLabels(r.Context(), ws.ID, rows, cols)
	}

	if len(selectedCols) > 0 {
		for i := range rows {
			rows[i].Data = applyDataSelect(rows[i].Data, selectedCols)
		}
	}

	httplib.Respond(w, http.StatusOK, models.NewPagedResult(rows, total, paging.Page, paging.PageSize))
}

// @Summary     Create row
// @Tags        rows
// @Accept      json
// @Produce     json
// @Param       wsCode     path      string         true  "workspace code"
// @Param       tableCode  path      string         true  "table code"
// @Param       body       body      RowDataRequest true  "row data"
// @Success     201        {object}  models.Row
// @Failure     400        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows [post]
func (h *RowHandler) Create(w http.ResponseWriter, r *http.Request) {
	t := httplib.TableFromCtx(r)
	var body RowDataRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httplib.BadRequest(w, "invalid request body")
		return
	}
	row, err := h.Svc.CreateRow(r.Context(), t.ID, body.Data)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	src := r.Header.Get("X-Command-Source-Id")
	cmd := r.Header.Get("X-Command-Id")
	h.Broker.Publish(t.ID, events.RowEvent{Type: events.RowEventCreated, Row: row, CommandSourceID: src, CommandID: cmd})
	httplib.Respond(w, http.StatusCreated, row)
}

// @Summary     Replace row
// @Tags        rows
// @Accept      json
// @Produce     json
// @Param       wsCode     path      string         true  "workspace code"
// @Param       tableCode  path      string         true  "table code"
// @Param       rowID      path      int            true  "row ID"
// @Param       body       body      RowDataRequest true  "row data"
// @Success     200        {object}  models.Row
// @Failure     400        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows/{rowID} [put]
func (h *RowHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "rowID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid row id")
		return
	}
	var body RowDataRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Data) == 0 {
		httplib.BadRequest(w, "data is required")
		return
	}
	t := httplib.TableFromCtx(r)
	if err := h.checkParentCycle(r.Context(), t.ID, t.Code, id, body.Data); err != nil {
		httplib.BadRequest(w, err.Error())
		return
	}
	row, revID, err := h.Svc.UpdateRow(r.Context(), t.ID, id, body.Data, r.Header.Get("X-Revision-Id"))
	if err != nil {
		if errors.Is(err, ErrRowNotFound) {
			httplib.NotFound(w)
			return
		}
		httplib.InternalErr(w, err)
		return
	}
	w.Header().Set("X-Revision-Id", revID)
	src := r.Header.Get("X-Command-Source-Id")
	cmd := r.Header.Get("X-Command-Id")
	h.Broker.Publish(t.ID, events.RowEvent{Type: events.RowEventUpdated, Row: row, CommandSourceID: src, CommandID: cmd})
	httplib.Respond(w, http.StatusOK, row)
}

// @Summary     Patch row
// @Tags        rows
// @Accept      json
// @Produce     json
// @Param       wsCode     path      string         true  "workspace code"
// @Param       tableCode  path      string         true  "table code"
// @Param       rowID      path      int            true  "row ID"
// @Param       body       body      RowDataRequest true  "partial row data"
// @Success     200        {object}  models.Row
// @Failure     400        {object}  http.ErrorResponse
// @Failure     404        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows/{rowID} [patch]
func (h *RowHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "rowID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid row id")
		return
	}
	var body RowDataRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Data) == 0 {
		httplib.BadRequest(w, "data is required")
		return
	}
	t := httplib.TableFromCtx(r)
	if err := h.checkParentCycle(r.Context(), t.ID, t.Code, id, body.Data); err != nil {
		httplib.BadRequest(w, err.Error())
		return
	}
	row, revID, err := h.Svc.PatchRow(r.Context(), t.ID, id, body.Data, r.Header.Get("X-Revision-Id"))
	if err != nil {
		if errors.Is(err, ErrRowNotFound) || errors.Is(err, sql.ErrNoRows) {
			httplib.NotFound(w)
			return
		}
		httplib.InternalErr(w, err)
		return
	}
	w.Header().Set("X-Revision-Id", revID)
	src := r.Header.Get("X-Command-Source-Id")
	cmd := r.Header.Get("X-Command-Id")
	h.Broker.Publish(t.ID, events.RowEvent{Type: events.RowEventUpdated, Row: row, CommandSourceID: src, CommandID: cmd})
	httplib.Respond(w, http.StatusOK, row)
}

// checkParentCycle returns an error if any self-referential row-link in
// patchData (code-keyed) would create a self-reference or immediate mutual
// reference (A→B→A). Longer cycles are allowed — the graph traversal handles
// them safely with its visited set.
func (h *RowHandler) checkParentCycle(ctx context.Context, tableID int64, tableCode string, rowID int64, patchData json.RawMessage) error {
	cols, err := h.Cols.ListColumns(ctx, tableID)
	if err != nil {
		return nil // non-fatal: skip check on error
	}
	var patch map[string]json.RawMessage
	if err := json.Unmarshal(patchData, &patch); err != nil {
		return nil
	}
	for _, col := range cols {
		if col.Type != models.ColumnTypeRowLink || col.Options == nil {
			continue
		}
		var opts struct {
			TargetTableCode string `json:"targetTableCode"`
		}
		if err := json.Unmarshal(col.Options, &opts); err != nil || opts.TargetTableCode != tableCode {
			continue
		}
		raw, ok := patch[col.Code]
		if !ok || string(raw) == "null" {
			continue
		}
		var link struct {
			ID int64 `json:"id"`
		}
		if err := json.Unmarshal(raw, &link); err != nil || link.ID == 0 {
			continue
		}
		// Self-reference: a row cannot be its own parent.
		if link.ID == rowID {
			return fmt.Errorf("circular parent reference detected")
		}
		// Mutual reference: does the proposed parent already point back to us?
		parent, err := h.Svc.GetRow(ctx, tableID, link.ID)
		if err != nil || parent == nil {
			continue
		}
		var parentData map[string]json.RawMessage
		if err := json.Unmarshal(parent.Data, &parentData); err != nil {
			continue
		}
		raw, ok = parentData[col.Code]
		if !ok || string(raw) == "null" {
			continue
		}
		var parentLink struct {
			ID int64 `json:"id"`
		}
		if err := json.Unmarshal(raw, &parentLink); err != nil || parentLink.ID == 0 {
			continue
		}
		if parentLink.ID == rowID {
			return fmt.Errorf("circular parent reference detected")
		}
	}
	return nil
}

func rowLabelByCode(data json.RawMessage, code string) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	raw, ok := m[code]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

func mergeJSON(src, patch json.RawMessage) (json.RawMessage, error) {
	base := make(map[string]json.RawMessage)
	if err := json.Unmarshal(src, &base); err != nil {
		return nil, err
	}
	overlay := make(map[string]json.RawMessage)
	if err := json.Unmarshal(patch, &overlay); err != nil {
		return nil, err
	}
	maps.Copy(base, overlay)
	return json.Marshal(base)
}

// @Summary     Get row
// @Tags        rows
// @Produce     json
// @Param       wsCode     path      string  true  "workspace code"
// @Param       tableCode  path      string  true   "table code"
// @Param       rowID      path      int     true   "row ID"
// @Param       select     query     string  false  "comma-separated column codes to include in data; omit to return all columns"
// @Success     200        {object}  models.Row
// @Failure     400        {object}  http.ErrorResponse
// @Failure     404        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows/{rowID} [get]
func (h *RowHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "rowID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid row id")
		return
	}
	t := httplib.TableFromCtx(r)
	row, err := h.Svc.GetRow(r.Context(), t.ID, id)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if row == nil {
		httplib.NotFound(w)
		return
	}

	if ws := httplib.WorkspaceFromCtx(r); ws != nil {
		cols, _ := h.Cols.ListColumns(r.Context(), t.ID)
		rows := []models.Row{*row}
		h.resolveRowLinkLabels(r.Context(), ws.ID, rows, cols)
		row = &rows[0]
	}

	if sel := r.URL.Query().Get("select"); sel != "" {
		var cols []string
		for _, s := range strings.Split(sel, ",") {
			if s = strings.TrimSpace(s); s != "" {
				cols = append(cols, s)
			}
		}
		if len(cols) > 0 {
			row.Data = applyDataSelect(row.Data, cols)
		}
	}
	httplib.Respond(w, http.StatusOK, row)
}

// @Summary     Bulk patch rows
// @Tags        rows
// @Accept      json
// @Produce     json
// @Param       wsCode     path      string           true  "workspace code"
// @Param       tableCode  path      string           true  "table code"
// @Param       body       body      BulkPatchRequest true  "filters, data, and optional preview flag"
// @Success     200        {object}  BulkPatchResponse
// @Failure     400        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows/bulk [post]
func (h *RowHandler) BulkPatch(w http.ResponseWriter, r *http.Request) {
	t := httplib.TableFromCtx(r)
	var body BulkPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httplib.BadRequest(w, "invalid request body")
		return
	}
	if len(body.Data) == 0 && !body.Preview {
		httplib.BadRequest(w, "data is required")
		return
	}
	updated, count, err := h.Svc.BulkPatch(r.Context(), t.ID, body.Filters, body.Data, body.Preview)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if body.Preview {
		httplib.Respond(w, http.StatusOK, BulkPatchResponse{Count: count})
		return
	}
	src := r.Header.Get("X-Command-Source-Id")
	cmd := r.Header.Get("X-Command-Id")
	for _, row := range updated {
		h.Broker.Publish(t.ID, events.RowEvent{Type: events.RowEventUpdated, Row: row, CommandSourceID: src, CommandID: cmd})
	}
	httplib.Respond(w, http.StatusOK, BulkPatchResponse{Updated: count})
}

// @Summary     Delete row
// @Tags        rows
// @Param       wsCode     path  string  true  "workspace code"
// @Param       tableCode  path  string  true  "table code"
// @Param       rowID      path  int     true  "row ID"
// @Success     204
// @Failure     400  {object}  http.ErrorResponse
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows/{rowID} [delete]
func (h *RowHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "rowID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid row id")
		return
	}
	t := httplib.TableFromCtx(r)
	row, err := h.Svc.GetRow(r.Context(), t.ID, id)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if row == nil {
		httplib.NotFound(w)
		return
	}
	if ws := httplib.WorkspaceFromCtx(r); ws != nil {
		refs, err := h.Store.FindRowLinkReferences(r.Context(), ws.ID, id)
		if err != nil {
			httplib.InternalErr(w, err)
			return
		}
		if len(refs) > 0 {
			httplib.Respond(w, http.StatusConflict, map[string]any{
				"error":      "This record is referenced by other records and cannot be deleted.",
				"code":       "referenced",
				"references": refs,
			})
			return
		}
	}
	if h.Files != nil {
		h.Files.DeleteRowFiles(r.Context(), id)
	}
	if err := h.Store.DeleteRow(r.Context(), id); err != nil {
		httplib.InternalErr(w, err)
		return
	}
	src := r.Header.Get("X-Command-Source-Id")
	cmd := r.Header.Get("X-Command-Id")
	h.Broker.Publish(t.ID, events.RowEvent{Type: events.RowEventDeleted, ID: id, CommandSourceID: src, CommandID: cmd})
	w.WriteHeader(http.StatusNoContent)
}

// @Summary     Row history
// @Tags        rows
// @Produce     json
// @Param       wsCode     path      string  true  "workspace code"
// @Param       tableCode  path      string  true  "table code"
// @Param       rowID      path      int     true  "row ID"
// @Success     200        {array}   models.RowHistory
// @Failure     400        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows/{rowID}/history [get]
func (h *RowHandler) History(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "rowID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid row id")
		return
	}
	t := httplib.TableFromCtx(r)
	entries, err := h.Svc.ListRowHistory(r.Context(), t.ID, id)
	if err != nil {
		if errors.Is(err, ErrRowNotFound) {
			httplib.NotFound(w)
			return
		}
		httplib.InternalErr(w, err)
		return
	}
	if entries == nil {
		entries = []models.RowHistory{}
	}
	httplib.Respond(w, http.StatusOK, entries)
}

type annotationBody struct {
	Annotation string `json:"annotation"`
}

// @Summary     Create annotation
// @Tags        rows
// @Accept      json
// @Produce     json
// @Param       wsCode     path      string          true  "workspace code"
// @Param       tableCode  path      string          true  "table code"
// @Param       rowID      path      int             true  "row ID"
// @Param       body       body      annotationBody  true  "annotation text"
// @Success     201        {object}  models.RowHistory
// @Failure     400        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows/{rowID}/history [post]
func (h *RowHandler) CreateAnnotation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "rowID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid row id")
		return
	}
	var body annotationBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httplib.BadRequest(w, "invalid body")
		return
	}
	if body.Annotation == "" {
		httplib.BadRequest(w, "annotation text is required")
		return
	}
	t := httplib.TableFromCtx(r)
	entry, err := h.Svc.CreateAnnotation(r.Context(), t.ID, id, body.Annotation)
	if err != nil {
		if errors.Is(err, ErrRowNotFound) {
			httplib.NotFound(w)
			return
		}
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, http.StatusCreated, entry)
}

// @Summary     Update annotation or change note
// @Tags        rows
// @Accept      json
// @Produce     json
// @Param       wsCode     path      string          true  "workspace code"
// @Param       tableCode  path      string          true  "table code"
// @Param       rowID      path      int             true  "row ID"
// @Param       histID     path      int             true  "history entry ID"
// @Param       body       body      annotationBody  true  "annotation text"
// @Success     200        {object}  models.RowHistory
// @Failure     400        {object}  http.ErrorResponse
// @Failure     404        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows/{rowID}/history/{histID} [patch]
func (h *RowHandler) UpdateHistoryAnnotation(w http.ResponseWriter, r *http.Request) {
	rowID, err := strconv.ParseInt(chi.URLParam(r, "rowID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid row id")
		return
	}
	histID, err := strconv.ParseInt(chi.URLParam(r, "histID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid history id")
		return
	}
	var body annotationBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httplib.BadRequest(w, "invalid body")
		return
	}
	t := httplib.TableFromCtx(r)
	// Try annotation entry first; fall back to change entry.
	entry, err := h.Svc.UpdateAnnotation(r.Context(), t.ID, rowID, histID, body.Annotation)
	if err != nil && !errors.Is(err, ErrRowNotFound) {
		entry, err = h.Svc.UpdateChangeAnnotation(r.Context(), t.ID, rowID, histID, body.Annotation)
	}
	if err != nil {
		httplib.NotFound(w)
		return
	}
	httplib.Respond(w, http.StatusOK, entry)
}

// @Summary     Delete annotation
// @Tags        rows
// @Param       wsCode     path  string  true  "workspace code"
// @Param       tableCode  path  string  true  "table code"
// @Param       rowID      path  int     true  "row ID"
// @Param       histID     path  int     true  "history entry ID"
// @Success     204
// @Failure     400  {object}  http.ErrorResponse
// @Failure     404  {object}  http.ErrorResponse
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows/{rowID}/history/{histID} [delete]
func (h *RowHandler) DeleteAnnotation(w http.ResponseWriter, r *http.Request) {
	rowID, err := strconv.ParseInt(chi.URLParam(r, "rowID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid row id")
		return
	}
	histID, err := strconv.ParseInt(chi.URLParam(r, "histID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid history id")
		return
	}
	t := httplib.TableFromCtx(r)
	if err := h.Svc.DeleteAnnotation(r.Context(), t.ID, rowID, histID); err != nil {
		httplib.NotFound(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
