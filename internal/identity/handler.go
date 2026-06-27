package identity

import (
	"encoding/json"
	"net/http"
	"strings"

	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/infrastructure/identifier"
	"github.com/saintedlama/degubase/internal/models"
)

// WorkspaceHandler handles workspace CRUD.
type WorkspaceHandler struct{ Store Store }

// @Summary     List workspaces
// @Tags        workspaces
// @Produce     json
// @Success     200  {array}   models.Workspace
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces [get]
func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	ws, err := h.Store.ListWorkspaces(r.Context())
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if ws == nil {
		ws = []models.Workspace{}
	}
	httplib.Respond(w, http.StatusOK, ws)
}

// @Summary     Create workspace
// @Tags        workspaces
// @Accept      json
// @Produce     json
// @Param       body    body      CreateWorkspaceRequest  true  "workspace"
// @Success     201     {object}  models.Workspace
// @Failure     400     {object}  http.ErrorResponse
// @Failure     500     {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces [post]
func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body CreateWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		httplib.BadRequest(w, "name is required")
		return
	}
	code := strings.ToUpper(body.Code)
	if code == "" {
		code = identifier.GenerateCode(body.Name)
	}
	ws, err := h.Store.CreateWorkspace(r.Context(), body.Name, body.Context, code)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, http.StatusCreated, ws)
}

// @Summary     Get workspace
// @Tags        workspaces
// @Produce     json
// @Param       wsCode  path      string  true  "workspace code"
// @Success     200     {object}  models.Workspace
// @Failure     404     {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode} [get]
func (h *WorkspaceHandler) Get(w http.ResponseWriter, r *http.Request) {
	httplib.Respond(w, http.StatusOK, httplib.WorkspaceFromCtx(r))
}

// @Summary     Update workspace
// @Tags        workspaces
// @Accept      json
// @Produce     json
// @Param       wsCode  path      string                  true  "workspace code"
// @Param       body    body      CreateWorkspaceRequest  true  "workspace"
// @Success     200     {object}  models.Workspace
// @Failure     400     {object}  http.ErrorResponse
// @Failure     500     {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode} [put]
func (h *WorkspaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	var body CreateWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		httplib.BadRequest(w, "name is required")
		return
	}
	ws := httplib.WorkspaceFromCtx(r)
	updated, err := h.Store.UpdateWorkspace(r.Context(), ws.ID, body.Name, body.Context)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	httplib.Respond(w, http.StatusOK, updated)
}

// @Summary     Delete workspace
// @Tags        workspaces
// @Param       wsCode  path  string  true  "workspace code"
// @Success     204
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode} [delete]
func (h *WorkspaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ws := httplib.WorkspaceFromCtx(r)
	if err := h.Store.DeleteWorkspace(r.Context(), ws.ID); err != nil {
		httplib.InternalErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
