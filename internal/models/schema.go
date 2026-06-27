package models

import (
	"encoding/json"
	"time"
)

type Table struct {
	ID            int64     `json:"id"`
	Code          string    `json:"code"`
	WorkspaceID   int64     `json:"workspace_id"`
	Name          string    `json:"name"`
	Context       string    `json:"context"`
	Icon          string    `json:"icon"`
	DefaultViewID *int64    `json:"default_view_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	Views         []View    `json:"views"`
	Columns       []Column  `json:"columns"`
}

type ColumnType string

const (
	ColumnTypeText         ColumnType = "text"
	ColumnTypeLongText     ColumnType = "long-text"
	ColumnTypeMarkdown     ColumnType = "markdown"
	ColumnTypeEmail        ColumnType = "email"
	ColumnTypeURL          ColumnType = "url"
	ColumnTypeNumber       ColumnType = "number"
	ColumnTypeCurrency     ColumnType = "currency"
	ColumnTypePercent      ColumnType = "percent"
	ColumnTypeRating       ColumnType = "rating"
	ColumnTypeDate         ColumnType = "date"
	ColumnTypeDatetime     ColumnType = "datetime"
	ColumnTypeCheckbox     ColumnType = "checkbox"
	ColumnTypeSingleSelect ColumnType = "single-select"
	ColumnTypeMultiSelect  ColumnType = "multi-select"
	ColumnTypeFile         ColumnType = "file"
	ColumnTypeImage        ColumnType = "image"
	ColumnTypeSymbol       ColumnType = "symbol"
	ColumnTypeEmoji        ColumnType = "emoji"
	ColumnTypeCreatedAt    ColumnType = "created-at"
	ColumnTypeUpdatedAt    ColumnType = "updated-at"
	ColumnTypeChecklist    ColumnType = "checklist"
	ColumnTypeRowLink      ColumnType = "row-link"
	ColumnTypeMermaid      ColumnType = "mermaid"
)

func (ct ColumnType) Valid() bool {
	switch ct {
	case ColumnTypeText, ColumnTypeLongText, ColumnTypeMarkdown, ColumnTypeEmail,
		ColumnTypeURL, ColumnTypeNumber, ColumnTypeCurrency, ColumnTypePercent,
		ColumnTypeRating, ColumnTypeDate, ColumnTypeDatetime, ColumnTypeCheckbox,
		ColumnTypeSingleSelect, ColumnTypeMultiSelect,
		ColumnTypeFile, ColumnTypeImage, ColumnTypeSymbol, ColumnTypeEmoji,
		ColumnTypeCreatedAt, ColumnTypeUpdatedAt, ColumnTypeChecklist, ColumnTypeRowLink,
		ColumnTypeMermaid:
		return true
	}
	return false
}

type Column struct {
	ID      int64      `json:"id"`
	Code    string     `json:"code"`
	TableID int64      `json:"table_id"`
	Name    string     `json:"name"`
	Type    ColumnType `json:"type"`
	// Options holds column-type-specific configuration. For single-select and multi-select columns:
	// {"choices":["A","B","C"],"choiceColors":{"A":"#e41a1c","B":"#377eb8"},"palette":0}.
	// Null or omitted for all other column types.
	Options  json.RawMessage `json:"options,omitempty"  swaggertype:"object"`
	Position int             `json:"position"`
}

// AppendVirtualColumns adds synthetic created_at/updated_at columns when absent.
// Virtual columns carry sentinel IDs (-1, -2) and are never persisted.
func AppendVirtualColumns(tableID int64, cols []Column) []Column {
	hasCreatedAt, hasUpdatedAt := false, false
	for _, c := range cols {
		if c.Type == ColumnTypeCreatedAt {
			hasCreatedAt = true
		}
		if c.Type == ColumnTypeUpdatedAt {
			hasUpdatedAt = true
		}
	}
	if !hasCreatedAt {
		cols = append(cols, Column{ID: -1, Code: "created_at", TableID: tableID, Name: "Created At", Type: ColumnTypeCreatedAt, Position: 9998})
	}
	if !hasUpdatedAt {
		cols = append(cols, Column{ID: -2, Code: "updated_at", TableID: tableID, Name: "Updated At", Type: ColumnTypeUpdatedAt, Position: 9999})
	}
	return cols
}

type ViewType string

const (
	ViewTypeTabular  ViewType = "tabular"
	ViewTypeKanban   ViewType = "kanban"
	ViewTypeCard     ViewType = "card"
	ViewTypeMatrix   ViewType = "matrix"
	ViewTypeTimeline ViewType = "timeline"
)

type CardFieldConfig struct {
	Col       string `json:"col"`
	ShowLabel bool   `json:"showLabel"`
	Truncate  bool   `json:"truncate"`
	Style     string `json:"style,omitempty"`
}

// ViewConfig holds the persisted display settings for a view.
// All column references use codes, never numeric IDs.
//
// Sub-configs are embedded by view type. Go's JSON encoder flattens them
// so the wire format stays unchanged — the frontend sees the same flat object.
type ViewConfig struct {
	// ── Shared across all views ────────────────────────────────────────────
	Sort        []RowSort         `json:"sort,omitempty"`
	Filters     []RowFilter       `json:"filters,omitempty"`
	GroupByCol  string            `json:"groupByCol,omitempty"`
	CardFields  []CardFieldConfig `json:"cardFields,omitempty"`
	ColumnOrder []string          `json:"columnOrder,omitempty"`
	XCol        string            `json:"xCol,omitempty"` // used by kanban (lanes) and matrix (columns)

	// ── View-type-specific configs (embedded — flattened in JSON) ──────────
	*TabularConfig
	*KanbanConfig
	*MatrixConfig
	*TimelineConfig
}

// TabularConfig holds settings specific to the tabular (grid) view.
type TabularConfig struct {
	HiddenCols []string `json:"hiddenCols,omitempty"` // column codes to hide
}

// KanbanConfig holds settings specific to the kanban (board) view.
type KanbanConfig struct {
	HiddenKanbanColumns []string `json:"hiddenKanbanColumns,omitempty"` // lane values to hide
}

// MatrixConfig holds settings specific to the matrix view.
type MatrixConfig struct {
	YCol           string `json:"yCol,omitempty"`
	HideAxisLabels bool   `json:"hideAxisLabels,omitempty"`
}

// TimelineConfig holds settings specific to the timeline/calendar view.
type TimelineConfig struct {
	DateCol         string `json:"dateCol,omitempty"`
	TimelineZoom    string `json:"timelineZoom,omitempty"`
	VisibleWeekdays []int  `json:"visibleWeekdays,omitempty"`
}

type View struct {
	ID        int64       `json:"id"`
	Code      string      `json:"code"`
	TableID   int64       `json:"table_id"`
	Name      string      `json:"name"`
	Type      ViewType    `json:"type"`
	Config    *ViewConfig `json:"config,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}
