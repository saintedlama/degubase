package automation

import "github.com/saintedlama/degubase/internal/models"

type CreateScriptRequest struct {
	Name      string                 `json:"name"`
	EventType models.ScriptEventType `json:"event_type"`
	Code      string                 `json:"code"`
	Enabled   bool                   `json:"enabled"`
	TableIDs  []int64                `json:"table_ids"`
}

type UpdateScriptRequest struct {
	Name      string                 `json:"name"`
	EventType models.ScriptEventType `json:"event_type"`
	Code      string                 `json:"code"`
	Enabled   bool                   `json:"enabled"`
	TableIDs  []int64                `json:"table_ids"`
}

type UpsertEnvRequest struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"is_secret"`
}
