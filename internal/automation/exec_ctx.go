package automation

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
)

// WithScriptExecution returns a context that marks the current call as
// originating from a script execution. Row writes carrying this context
// must not dispatch further scripts (cycle-prevention contract). The
// commandSourceID for SSE is derived as "script-{execID}" — stable for
// the lifetime of that execution and namespaced away from browser tabs
// ("ui-{uuid}").
func WithScriptExecution(ctx context.Context, execID int64) context.Context {
	return context.WithValue(ctx, httplib.CtxScriptExec, execID)
}

// scriptExecutionID returns the execution ID from the context, or 0 when
// the call did not originate from a script.
func ScriptExecutionID(ctx context.Context) int64 {
	id, _ := ctx.Value(httplib.CtxScriptExec).(int64)
	return id
}

// commandSourceFromCtx returns the (commandSourceID, commandID) pair for a
// broker publish. For normal HTTP requests it reads the headers; for internal
// script write-backs it derives a stable "script-{execID}" source and a fresh
// command ID so SSE events are correctly namespaced.
func CommandSourceFromCtx(ctx context.Context, headerSource, headerCmd string) (source, cmd string) {
	if headerSource != "" {
		return headerSource, headerCmd
	}
	return fmt.Sprintf("script-%d", ScriptExecutionID(ctx)), uuid.NewString()
}
