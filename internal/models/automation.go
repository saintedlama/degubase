package models

import "time"

type ScriptEventType string

const (
	ScriptEventRecordCreated ScriptEventType = "record.created"
	ScriptEventRecordUpdated ScriptEventType = "record.updated"
	ScriptEventRecordDeleted ScriptEventType = "record.deleted"
)

func (et ScriptEventType) Valid() bool {
	switch et {
	case ScriptEventRecordCreated, ScriptEventRecordUpdated, ScriptEventRecordDeleted:
		return true
	}
	return false
}

type Script struct {
	ID          int64           `json:"id"`
	WorkspaceID int64           `json:"workspace_id"`
	TableIDs    []int64         `json:"table_ids"`
	Name        string          `json:"name"`
	EventType   ScriptEventType `json:"event_type"`
	Code        string          `json:"code"`
	Enabled     bool            `json:"enabled"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type ScriptLogLevel string

const (
	ScriptLogInfo  ScriptLogLevel = "info"
	ScriptLogWarn  ScriptLogLevel = "warn"
	ScriptLogError ScriptLogLevel = "error"
)

type ScriptLogEntry struct {
	Level   ScriptLogLevel `json:"level"`
	Message string         `json:"message"`
	Ts      time.Time      `json:"ts"`
}

type ScriptExecution struct {
	ID            int64            `json:"id"`
	ScriptID      int64            `json:"script_id"`
	RowID         *int64           `json:"row_id,omitempty"`
	TableCode     string           `json:"table_code"`
	EventType     ScriptEventType  `json:"event_type"`
	StartedAt     time.Time        `json:"started_at"`
	EndedAt       time.Time        `json:"ended_at"`
	DurationMs    int64            `json:"duration_ms"`
	Success       bool             `json:"success"`
	Logs          []ScriptLogEntry `json:"logs"`
	Depth         int              `json:"depth"`
	TriggeredByID *int64           `json:"triggered_by_execution_id,omitempty"`
}

type ScriptEnvVar struct {
	ID          int64  `json:"id"`
	WorkspaceID int64  `json:"workspace_id"`
	Key         string `json:"key"`
	Value       string `json:"value,omitempty"`
	IsSecret    bool   `json:"is_secret"`
}
