package models

import (
	"encoding/json"
	"time"
)

type Row struct {
	ID        int64           `json:"id"`
	TableID   int64           `json:"table_id"`
	Data      json.RawMessage `json:"data"          swaggertype:"object"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type RowHistory struct {
	ID         int64           `json:"id"`
	RowID      int64           `json:"row_id"`
	EntryType  string          `json:"entry_type"` // "change" or "annotation"
	Data       json.RawMessage `json:"data,omitempty" swaggertype:"object"`
	Annotation string          `json:"annotation,omitempty"`
	ChangedAt  time.Time       `json:"changed_at"`
	RevisionID string          `json:"revision_id,omitempty"`
}

type RowFilter struct {
	Col   string `json:"col"`
	Op    string `json:"op"`
	Value string `json:"value"`
}

type RowSort struct {
	Col string `json:"col"`
	Dir string `json:"dir"` // "asc" or "desc"
}

type PagedResult[T any] struct {
	Data       []T   `json:"data"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

func NewPagedResult[T any](data []T, total int64, page, pageSize int) PagedResult[T] {
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}
	return PagedResult[T]{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

type PagedRows = PagedResult[Row]

type PagedReferencingRows = PagedResult[ReferencingRow]

// RowLinkRef identifies a row in another table that holds a reference to a
// row being deleted. Returned in 409 responses to blocked deletes.
type RowLinkRef struct {
	TableCode string `json:"tableCode"`
	TableName string `json:"tableName"`
	RowID     int64  `json:"rowId"`
}

// RowLinkEntry is a single row-link value extracted from row data for syncing
// into the row_links table. ColID and TargetRowID are numeric IDs (DB-internal).
type RowLinkEntry struct {
	ColID       int64
	TargetRowID int64
	TargetColID int64 // resolved display column on the target table, 0 if unresolvable
}

// RowGroup is one bucket returned by the grouped-rows endpoint.
// Value is nil for rows whose group column has no value set.
// Total is the full count for this group; Rows contains at most pageSize entries.
type RowGroup struct {
	Value *string `json:"value"`
	Total int64   `json:"total"`
	Rows  []Row   `json:"rows"`
}

// ReferencingRow describes a row that links to the current row via a row-link
// column. The "current row" is the target of the link.
type ReferencingRow struct {
	SourceRowID     int64  `json:"sourceRowId"`
	SourceTableCode string `json:"sourceTableCode"`
	SourceTableName string `json:"sourceTableName"`
	SourceRowLabel  string `json:"sourceRowLabel"`
	ViaColumnCode   string `json:"viaColumnCode"`
	ViaColumnName   string `json:"viaColumnName"`
}
