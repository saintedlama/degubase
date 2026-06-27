package schema

import (
	"encoding/json"

	"github.com/saintedlama/degubase/internal/models"
)

type CreateTableRequest struct {
	Name    string `json:"name"    example:"Customers"`
	Context string `json:"context" example:"Stores customer records"`
	Icon    string `json:"icon,omitempty" example:"ri-database-2-line"`
	Code    string `json:"code"    example:"CUST"`
}

type UpdateTableRequest struct {
	Name          string `json:"name"    example:"Customers"`
	Context       string `json:"context" example:"Stores customer records"`
	Icon          string `json:"icon,omitempty" example:"ri-database-2-line"`
	DefaultViewID *int64 `json:"default_view_id"`
}

type CreateColumnRequest struct {
	Name    string            `json:"name"    example:"Email"`
	Type    models.ColumnType `json:"type"    example:"email"`
	Options json.RawMessage   `json:"options" swaggertype:"object"`
}

type UpdateColumnRequest struct {
	Name    string            `json:"name"    example:"Email"`
	Type    models.ColumnType `json:"type"    example:"email"`
	Options json.RawMessage   `json:"options" swaggertype:"object"`
}

type ReorderColumnsRequest struct {
	Codes []string `json:"codes" example:"name,email,status"`
}

type CreateViewRequest struct {
	Name   string             `json:"name"   example:"Default"`
	Type   models.ViewType    `json:"type"   example:"tabular"`
	Config *models.ViewConfig `json:"config"`
}

type UpdateViewRequest struct {
	Name   string             `json:"name"   example:"Default"`
	Config *models.ViewConfig `json:"config"`
}
