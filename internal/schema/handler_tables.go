package schema

import (
	"encoding/json"
	nethttp "net/http"
	"strings"

	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/infrastructure/identifier"
	"github.com/saintedlama/degubase/internal/models"
)

type TableHandler struct{ Store Store }

// @Summary     List tables
// @Tags        tables
// @Produce     json
// @Param       wsCode  path      string  true  "workspace code"
// @Success     200     {array}   models.Table
// @Failure     500     {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables [get]
func (h *TableHandler) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	ws := httplib.WorkspaceFromCtx(r)
	tables, err := h.Store.ListTables(r.Context(), ws.ID)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if tables == nil {
		tables = []models.Table{}
	}
	for i := range tables {
		views, err := h.Store.ListViews(r.Context(), tables[i].ID)
		if err != nil {
			httplib.InternalErr(w, err)
			return
		}
		if views == nil {
			views = []models.View{}
		}
		tables[i].Views = views
	}
	httplib.Respond(w, nethttp.StatusOK, tables)
}

// @Summary     Create table
// @Tags        tables
// @Accept      json
// @Produce     json
// @Param       wsCode  path      string              true  "workspace code"
// @Param       body    body      CreateTableRequest  true  "table"
// @Success     201     {object}  models.Table
// @Failure     400     {object}  http.ErrorResponse
// @Failure     500     {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables [post]
func (h *TableHandler) Create(w nethttp.ResponseWriter, r *nethttp.Request) {
	ws := httplib.WorkspaceFromCtx(r)
	var body CreateTableRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		httplib.BadRequest(w, "name is required")
		return
	}
	code := strings.ToUpper(body.Code)
	if code == "" {
		code = identifier.GenerateCode(body.Name)
	}
	t, err := h.Store.CreateTable(r.Context(), ws.ID, body.Name, body.Context, body.Icon, code)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	full, err := h.embed(r, t)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, nethttp.StatusCreated, full)
}

// @Summary     Get table
// @Tags        tables
// @Produce     json
// @Param       wsCode     path      string  true  "workspace code"
// @Param       tableCode  path      string  true  "table code"
// @Success     200        {object}  models.Table
// @Failure     404        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode} [get]
func (h *TableHandler) Get(w nethttp.ResponseWriter, r *nethttp.Request) {
	t := httplib.TableFromCtx(r)
	full, err := h.embed(r, t)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, nethttp.StatusOK, full)
}

// @Summary     Update table
// @Tags        tables
// @Accept      json
// @Produce     json
// @Param       wsCode     path      string              true  "workspace code"
// @Param       tableCode  path      string              true  "table code"
// @Param       body       body      UpdateTableRequest  true  "table"
// @Success     200        {object}  models.Table
// @Failure     400        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode} [put]
func (h *TableHandler) Update(w nethttp.ResponseWriter, r *nethttp.Request) {
	t := httplib.TableFromCtx(r)
	var body UpdateTableRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		httplib.BadRequest(w, "name is required")
		return
	}
	updated, err := h.Store.UpdateTable(r.Context(), t.ID, body.Name, body.Context, body.Icon, body.DefaultViewID)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	full, err := h.embed(r, updated)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, nethttp.StatusOK, full)
}

// @Summary     Delete table
// @Tags        tables
// @Param       wsCode     path  string  true  "workspace code"
// @Param       tableCode  path  string  true  "table code"
// @Success     204
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode} [delete]
func (h *TableHandler) Delete(w nethttp.ResponseWriter, r *nethttp.Request) {
	t := httplib.TableFromCtx(r)
	if err := h.Store.DeleteTable(r.Context(), t.ID); err != nil {
		httplib.InternalErr(w, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}

func (h *TableHandler) embed(r *nethttp.Request, t *models.Table) (*models.Table, error) {
	cols, err := h.Store.ListColumns(r.Context(), t.ID)
	if err != nil {
		return nil, err
	}
	if cols == nil {
		cols = []models.Column{}
	}
	t.Columns = models.AppendVirtualColumns(t.ID, cols)

	views, err := h.Store.ListViews(r.Context(), t.ID)
	if err != nil {
		return nil, err
	}
	if views == nil {
		views = []models.View{}
	}
	t.Views = views

	return t, nil
}
