package http

import (
	nethttp "net/http"

	"github.com/saintedlama/degubase/internal/models"
)

type ctxKey int

const (
	CtxWorkspace ctxKey = iota
	CtxTable
	CtxScriptExec
	CtxUser
	CtxTokenWsID
)

func WorkspaceFromCtx(r *nethttp.Request) *models.Workspace {
	ws, _ := r.Context().Value(CtxWorkspace).(*models.Workspace)
	return ws
}

func TableFromCtx(r *nethttp.Request) *models.Table {
	tbl, _ := r.Context().Value(CtxTable).(*models.Table)
	return tbl
}

func UserFromCtx(r *nethttp.Request) *models.User {
	u, _ := r.Context().Value(CtxUser).(*models.User)
	return u
}

func TokenWsIDFromCtx(r *nethttp.Request) (int64, bool) {
	id, ok := r.Context().Value(CtxTokenWsID).(int64)
	return id, ok
}
