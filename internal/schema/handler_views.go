package schema

import (
	"encoding/json"
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/models"
)

type ViewHandler struct{ Store Store }

// @Summary     List views
// @Tags        views
// @Produce     json
// @Param       wsCode     path      string  true  "workspace code"
// @Param       tableCode  path      string  true  "table code"
// @Success     200        {array}   models.View
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/views [get]
func (h *ViewHandler) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	t := httplib.TableFromCtx(r)
	views, err := h.Store.ListViews(r.Context(), t.ID)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if views == nil {
		views = []models.View{}
	}
	httplib.Respond(w, nethttp.StatusOK, views)
}

// @Summary     Create view
// @Tags        views
// @Accept      json
// @Produce     json
// @Param       wsCode     path      string             true  "workspace code"
// @Param       tableCode  path      string             true  "table code"
// @Param       body       body      CreateViewRequest  true  "view"
// @Success     201        {object}  models.View
// @Failure     400        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/views [post]
func (h *ViewHandler) Create(w nethttp.ResponseWriter, r *nethttp.Request) {
	t := httplib.TableFromCtx(r)
	var body CreateViewRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		httplib.BadRequest(w, "name is required")
		return
	}
	if body.Type != models.ViewTypeTabular && body.Type != models.ViewTypeKanban && body.Type != models.ViewTypeCard && body.Type != models.ViewTypeMatrix && body.Type != models.ViewTypeTimeline {
		httplib.BadRequest(w, "type must be tabular, kanban, card, matrix, or timeline")
		return
	}
	v, err := h.Store.CreateView(r.Context(), t.ID, body.Name, body.Type, body.Config)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, nethttp.StatusCreated, v)
}

// @Summary     Get view
// @Tags        views
// @Produce     json
// @Param       wsCode     path      string  true  "workspace code"
// @Param       tableCode  path      string  true  "table code"
// @Param       viewCode   path      string  true  "view code"
// @Success     200        {object}  models.View
// @Failure     404        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/views/{viewCode} [get]
func (h *ViewHandler) Get(w nethttp.ResponseWriter, r *nethttp.Request) {
	t := httplib.TableFromCtx(r)
	v, err := h.Store.GetViewByCode(r.Context(), t.ID, chi.URLParam(r, "viewCode"))
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if v == nil {
		httplib.NotFound(w)
		return
	}
	httplib.Respond(w, nethttp.StatusOK, v)
}

// @Summary     Update view
// @Tags        views
// @Accept      json
// @Produce     json
// @Param       wsCode     path      string             true  "workspace code"
// @Param       tableCode  path      string             true  "table code"
// @Param       viewCode   path      string             true  "view code"
// @Param       body       body      UpdateViewRequest  true  "view"
// @Success     200        {object}  models.View
// @Failure     400        {object}  http.ErrorResponse
// @Failure     404        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/views/{viewCode} [put]
func (h *ViewHandler) Update(w nethttp.ResponseWriter, r *nethttp.Request) {
	t := httplib.TableFromCtx(r)
	v, err := h.Store.GetViewByCode(r.Context(), t.ID, chi.URLParam(r, "viewCode"))
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if v == nil {
		httplib.NotFound(w)
		return
	}
	var body UpdateViewRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		httplib.BadRequest(w, "name is required")
		return
	}
	updated, err := h.Store.UpdateView(r.Context(), v.ID, body.Name, body.Config)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, nethttp.StatusOK, updated)
}

// @Summary     Delete view
// @Tags        views
// @Param       wsCode     path  string  true  "workspace code"
// @Param       tableCode  path  string  true  "table code"
// @Param       viewCode   path  string  true  "view code"
// @Success     204
// @Failure     400  {object}  http.ErrorResponse
// @Failure     404  {object}  http.ErrorResponse
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/views/{viewCode} [delete]
func (h *ViewHandler) Delete(w nethttp.ResponseWriter, r *nethttp.Request) {
	t := httplib.TableFromCtx(r)
	v, err := h.Store.GetViewByCode(r.Context(), t.ID, chi.URLParam(r, "viewCode"))
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if v == nil {
		httplib.NotFound(w)
		return
	}
	if err := h.Store.DeleteView(r.Context(), v.ID); err != nil {
		httplib.RespondErr(w, nethttp.StatusBadRequest, "delete_failed", err.Error())
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}
