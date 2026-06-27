// Package automation manages scripts, env vars, script executions, and the Lua runtime.
package automation

import (
	"context"
	"encoding/json"

	"github.com/saintedlama/degubase/internal/models"
)

// ScriptStore is the script persistence contract.
type ScriptStore interface {
	ListScripts(ctx context.Context, workspaceID int64) ([]models.Script, error)
	CreateScript(ctx context.Context, workspaceID int64, tableIDs []int64, name string, eventType models.ScriptEventType, code string, enabled bool) (*models.Script, error)
	GetScript(ctx context.Context, id int64) (*models.Script, error)
	UpdateScript(ctx context.Context, id int64, tableIDs []int64, name string, eventType models.ScriptEventType, code string, enabled bool) (*models.Script, error)
	DeleteScript(ctx context.Context, id int64) error
}

// ScriptEnvStore is the script environment variable persistence contract.
type ScriptEnvStore interface {
	ListScriptEnvVars(ctx context.Context, workspaceID int64) ([]models.ScriptEnvVar, error)
	UpsertScriptEnvVar(ctx context.Context, workspaceID int64, key, value string, isSecret bool) (*models.ScriptEnvVar, error)
	DeleteScriptEnvVar(ctx context.Context, id int64) error
}

// ScriptExecutionStore is the script execution history persistence contract.
type ScriptExecutionStore interface {
	ListScriptExecutions(ctx context.Context, scriptID int64, limit int) ([]models.ScriptExecution, error)
	CreateScriptExecution(ctx context.Context, exec models.ScriptExecution) (*models.ScriptExecution, error)
}

// Store composes all automation-scoped stores.
type Store interface {
	ScriptStore
	ScriptEnvStore
	ScriptExecutionStore
}

// RowMutator is the narrow records dependency needed for Lua script execution.
type RowMutator interface {
	GetRow(ctx context.Context, id int64) (*models.Row, error)
	CreateRow(ctx context.Context, tableID int64, data json.RawMessage) (*models.Row, error)
	UpdateRow(ctx context.Context, id int64, data json.RawMessage) (*models.Row, error)
}

// ColumnLister is the narrow schema dependency needed for Lua column name translation.
type ColumnLister interface {
	ListColumns(ctx context.Context, tableID int64) ([]models.Column, error)
}
