package models

import "time"

type Skill struct {
	ID          int64     `json:"id"`
	WorkspaceID int64     `json:"workspace_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	TableIDs    []int64   `json:"table_ids"`
	Operations  []string  `json:"operations"`
	TokenID     *int64    `json:"token_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
