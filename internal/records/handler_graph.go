package records

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/go-chi/chi/v5"
	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/models"
	"github.com/saintedlama/degubase/internal/schema"
)

// GraphNode is a vertex in the row-link graph.
type GraphNode struct {
	// ID uniquely identifies the node as "{tableCode}:{rowID}".
	ID    string `json:"id"`
	Table string `json:"table"`
	RowID int64  `json:"row_id"`
	// Label is the best human-readable name for the row (first text column, or "Row #N").
	// Deleted/orphaned rows carry their stale label with a " (deleted)" deleted suffix.
	Label string `json:"label"`
	// Data holds the code-keyed cell data. Null for orphaned (deleted) rows.
	Data json.RawMessage `json:"data" swaggertype:"object"`
}

// GraphEdge is a directed relationship between two nodes.
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	// ViaColumn is the row-link column code on the source row.
	ViaColumn     string `json:"via_column"`
	ViaColumnName string `json:"via_column_name"`
}

// GraphResponse is the full graph rooted at one record.
type GraphResponse struct {
	// Root is the node ID of the starting record.
	Root  string      `json:"root"`
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GraphHandler builds row-link relationship graphs.
type GraphHandler struct {
	Svc  *RowService
	Tbls schema.TableStore
}

// graphQueueItem is one BFS frontier entry.
type graphQueueItem struct {
	tableCode string
	tableID   int64
	rowID     int64
	depth     int
}

const (
	defaultGraphDepth = 5
	maxGraphDepth     = 10
)

// @Summary     Build relationship graph for a row
// @Description Traverses all row-link columns reachable from the given row up to the
// @Description specified depth and returns a graph with nodes and directed edges.
// @Description Cyclic references (A→B→C→A) are handled safely — each node appears exactly once.
// @Description Orphaned links (the referenced row was deleted) produce a ghost node so the edge remains visible.
// @Tags        rows
// @Produce     json
// @Param       wsCode     path      string  true   "workspace code"
// @Param       tableCode  path      string  true   "table code"
// @Param       rowID      path      int     true   "row ID"
// @Param       depth      query     int     false  "max traversal depth (default 5, max 10)"
// @Success     200        {object}  records.GraphResponse
// @Failure     400        {object}  http.ErrorResponse
// @Failure     404        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows/{rowID}/graph [get]
func (h *GraphHandler) Graph(w http.ResponseWriter, r *http.Request) {
	rowID, err := strconv.ParseInt(chi.URLParam(r, "rowID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid row id")
		return
	}

	depth := defaultGraphDepth
	if d := r.URL.Query().Get("depth"); d != "" {
		if n, err := strconv.Atoi(d); err == nil {
			depth = n
		}
	}
	if depth < 1 {
		depth = 1
	}
	if depth > maxGraphDepth {
		depth = maxGraphDepth
	}

	ws := httplib.WorkspaceFromCtx(r)
	t := httplib.TableFromCtx(r)

	// Verify root row exists.
	root, err := h.Svc.GetRow(r.Context(), t.ID, rowID)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if root == nil {
		httplib.NotFound(w)
		return
	}

	resp, err := h.buildGraph(r, ws, t, rowID, depth)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, http.StatusOK, resp)
}

func (h *GraphHandler) buildGraph(r *http.Request, ws *models.Workspace, rootTable *models.Table, rootRowID int64, maxDepth int) (*GraphResponse, error) {
	ctx := r.Context()
	rootID := graphNodeID(rootTable.Code, rootRowID)

	visited := map[string]bool{rootID: true}
	var nodes []GraphNode
	var edges []GraphEdge

	queue := []graphQueueItem{{rootTable.Code, rootTable.ID, rootRowID, 0}}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		row, err := h.Svc.GetRow(ctx, item.tableID, item.rowID)
		if err != nil {
			return nil, err
		}

		currentID := graphNodeID(item.tableCode, item.rowID)

		// A nil row or a row whose table_id doesn't match means the row was
		// deleted (and SQLite may have reused the ID for a row in another table).
		if row == nil || row.TableID != item.tableID {
			nodes = append(nodes, GraphNode{
				ID:    currentID,
				Table: item.tableCode,
				RowID: item.rowID,
				Label: fmt.Sprintf("Row #%d (deleted)", item.rowID),
				Data:  json.RawMessage("null"),
			})
			continue
		}

		// Fetch columns for this table.
		cols, err := h.Svc.Cols.ListColumns(ctx, item.tableID)
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, GraphNode{
			ID:    currentID,
			Table: item.tableCode,
			RowID: item.rowID,
			Label: graphRowLabel(row, cols),
			Data:  row.Data,
		})

		if item.depth >= maxDepth {
			continue
		}

		// Parse data once for link extraction.
		var dataMap map[string]json.RawMessage
		_ = json.Unmarshal(row.Data, &dataMap)

		for _, col := range cols {
			if col.Type != models.ColumnTypeRowLink || col.Options == nil {
				continue
			}

			var opts struct {
				TargetTableCode string `json:"targetTableCode"`
			}
			if err := json.Unmarshal(col.Options, &opts); err != nil || opts.TargetTableCode == "" {
				continue
			}

			// Extract the stored {id, label} value.
			cellRaw, ok := dataMap[col.Code]
			if !ok || len(cellRaw) == 0 || string(cellRaw) == "null" {
				continue
			}
			var link struct {
				ID    int64  `json:"id"`
				Label string `json:"label"`
			}
			if err := json.Unmarshal(cellRaw, &link); err != nil || link.ID == 0 {
				continue
			}

			toID := graphNodeID(opts.TargetTableCode, link.ID)

			edges = append(edges, GraphEdge{
				From:          currentID,
				To:            toID,
				ViaColumn:     col.Code,
				ViaColumnName: col.Name,
			})

			if visited[toID] {
				continue // cycle or already scheduled — do not re-enqueue
			}
			visited[toID] = true

			// Resolve target table.
			tgt, err := h.Tbls.GetTableByCode(ctx, ws.ID, opts.TargetTableCode)
			if err != nil {
				return nil, err
			}
			if tgt == nil {
				// Target table was deleted — add a ghost node for the edge target.
				nodes = append(nodes, GraphNode{
					ID:    toID,
					Table: opts.TargetTableCode,
					RowID: link.ID,
					Label: graphStaleLabel(link.Label, "table deleted"),
					Data:  json.RawMessage("null"),
				})
				continue
			}

			queue = append(queue, graphQueueItem{opts.TargetTableCode, tgt.ID, link.ID, item.depth + 1})
		}
	}

	// Ghost nodes for edges whose target rows were already marked visited but
	// were never fetched (they were skipped due to cycle detection, which is
	// correct — the node already exists). Nothing extra needed here.

	if nodes == nil {
		nodes = []GraphNode{}
	}
	if edges == nil {
		edges = []GraphEdge{}
	}
	return &GraphResponse{Root: rootID, Nodes: nodes, Edges: edges}, nil
}

// graphNodeID builds the unique node identifier: "{tableCode}:{rowID}".
func graphNodeID(tableCode string, rowID int64) string {
	return fmt.Sprintf("%s:%d", tableCode, rowID)
}

// graphRowLabel returns the first non-empty text-like column value, or "Row #N".
func graphRowLabel(row *models.Row, cols []models.Column) string {
	var dataMap map[string]json.RawMessage
	if err := json.Unmarshal(row.Data, &dataMap); err != nil || len(dataMap) == 0 {
		return fmt.Sprintf("Row #%d", row.ID)
	}

	textTypes := map[models.ColumnType]bool{
		models.ColumnTypeText:     true,
		models.ColumnTypeLongText: true,
		models.ColumnTypeMarkdown: true,
		models.ColumnTypeEmail:    true,
		models.ColumnTypeURL:      true,
	}

	sorted := make([]models.Column, len(cols))
	copy(sorted, cols)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Position < sorted[j].Position })

	for _, col := range sorted {
		if !textTypes[col.Type] {
			continue
		}
		raw, ok := dataMap[col.Code]
		if !ok {
			continue
		}
		var s string
		if err := json.Unmarshal(raw, &s); err != nil || s == "" {
			continue
		}
		return s
	}
	return fmt.Sprintf("Row #%d", row.ID)
}

func graphStaleLabel(stale, reason string) string {
	if stale == "" {
		return fmt.Sprintf("(unknown — %s)", reason)
	}
	return fmt.Sprintf("%s (%s)", stale, reason)
}
