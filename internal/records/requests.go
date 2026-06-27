package records

import (
	"encoding/json"

	"github.com/saintedlama/degubase/internal/models"
)

// RowDataRequest is the body for creating or replacing a row.
type RowDataRequest struct {
	// Data maps column codes to typed values. Format by column type:
	// text/long-text/markdown/email/url/symbol/emoji: string;
	// number/currency/percent/rating: number;
	// checkbox: boolean;
	// date: ISO 8601 date string e.g. "2024-01-15";
	// datetime: ISO 8601 datetime string e.g. "2024-01-15T10:00:00Z";
	// single-select: string matching one of the column's choices;
	// multi-select: array of strings e.g. ["Bug","Feature"];
	// checklist: array of {text: string, checked: boolean} objects e.g. [{"text":"Step 1","checked":false}];
	// file/image: managed via the file upload endpoint, not set directly in row data.
	Data json.RawMessage `json:"data" swaggertype:"object"`
}

// BulkPatchRequest is the body for bulk patching rows.
type BulkPatchRequest struct {
	Filters []models.RowFilter `json:"filters"`
	Data    json.RawMessage    `json:"data" swaggertype:"object"`
	Preview bool               `json:"preview"`
}

// BulkPatchResponse is returned by the bulk patch endpoint.
type BulkPatchResponse struct {
	Count   int64 `json:"count"`
	Updated int64 `json:"updated,omitempty"`
}
